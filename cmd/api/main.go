package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/auth"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/importlistings"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/logactivity"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/readleads"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/infrastructure/postgres"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/scorestrategies"
	httptransport "github.com/EstateLinkAI/estatelink-lead-engine/internal/transport/http"
	

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := mustGetEnv("DATABASE_URL")
	jwtSecret := mustGetEnv("JWT_SECRET")

	accessTokenTTL := getDurationEnv("ACCESS_TOKEN_TTL", 24*time.Hour)
	refreshTokenTTL := getDurationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour)

	ctx := context.Background()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Repositories
	listingRepo := postgres.NewListingRepository(db)
	leadScoreRepo := postgres.NewLeadScoreRepository(db)
	strategyScoreRepo := postgres.NewPropertyStrategyScoreRepository(db)
	leadReadRepo := postgres.NewLeadReadRepository(db)
	userRepo := postgres.NewUserRepository(db)
	rawListingRepo := postgres.NewRawListingRepository(db)
	importJobRepo := postgres.NewImportJobRepository(db)
	activityLogRepo := postgres.NewActivityLogRepository(db)
	

	// Use cases / services
	strategyScorer := scorestrategies.NewUseCase(strategyScoreRepo)
	ingestUseCase := ingestlisting.NewUseCase(listingRepo, leadScoreRepo, strategyScorer)
	readLeadsUseCase := readleads.NewUseCase(leadReadRepo)

	passwordHasher := auth.NewPasswordHasher()
	tokenService := auth.NewTokenService(jwtSecret, accessTokenTTL, refreshTokenTTL)
	authService := auth.NewService(userRepo, passwordHasher, tokenService)

	activityService := &logactivity.Service{
		Repo: activityLogRepo,
	}

	importListingsUseCase := importlistings.NewUseCase(
		rawListingRepo,
		importJobRepo,
		ingestUseCase,
		activityService,
	)

	// Handlers
	listingHandler := httptransport.NewListingHandler(ingestUseCase, activityService)
	leadHandler := httptransport.NewLeadHandler(readLeadsUseCase)
	authHandler := httptransport.NewAuthHandler(authService, activityService)
	importHandler := httptransport.NewImportHandler(importListingsUseCase)
	activityHandler := httptransport.NewActivityLogHandler(activityService)

	// Router
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
		},
		ExposedHeaders: []string{
			"Link",
		},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Get("/health", healthHandler)

	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/refresh", authHandler.Refresh)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(httptransport.AuthMiddleware(tokenService))

		r.Get("/api/me", authHandler.Me)

		leadHandler.RegisterRoutes(r)
	})

	// Admin / analyst routes
	r.Group(func(r chi.Router) {
		r.Use(httptransport.AuthMiddleware(tokenService))
		r.Use(httptransport.RequireRole(user.RoleAdmin, user.RoleAnalyst))

		listingHandler.RegisterRoutes(r)

		r.Post("/api/imports/clean-listings", importHandler.ImportCleanListings)
		r.Get("/api/imports/{jobId}", importHandler.GetImportJob)
	})

	// Admin-only routes
	r.Group(func(r chi.Router) {
		r.Use(httptransport.AuthMiddleware(tokenService))
		r.Use(httptransport.RequireRole(user.RoleAdmin))

		r.Patch("/api/admin/users/{id}/role", authHandler.UpdateUserRole)

		activityHandler.RegisterRoutes(r)
	})

	log.Println("server running on :8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func mustGetEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s environment variable is required", name)
	}

	return value
}

func getDurationEnv(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("%s must be a valid Go duration, got %q: %v", name, value, err)
	}

	return duration
}