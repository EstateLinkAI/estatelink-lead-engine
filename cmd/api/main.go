package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/auth"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/importlistings"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/readleads"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/infrastructure/postgres"
	httptransport "github.com/EstateLinkAI/estatelink-lead-engine/internal/transport/http"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	listingRepo := postgres.NewListingRepository(db)
	leadScoreRepo := postgres.NewLeadScoreRepository(db)
	leadReadRepo := postgres.NewLeadReadRepository(db)
	userRepo := postgres.NewUserRepository(db)
	rawListingRepo := postgres.NewRawListingRepository(db)
	importJobRepo := postgres.NewImportJobRepository(db)

	ingestUseCase := ingestlisting.NewUseCase(listingRepo, leadScoreRepo)
	readLeadsUseCase := readleads.NewUseCase(leadReadRepo)
	importListingsUseCase := importlistings.NewUseCase(rawListingRepo, importJobRepo, ingestUseCase)

	passwordHasher := auth.NewPasswordHasher()
	tokenService := auth.NewTokenService(jwtSecret, 24*time.Hour)
	authService := auth.NewService(userRepo, passwordHasher, tokenService)

	listingHandler := httptransport.NewListingHandler(ingestUseCase)
	leadHandler := httptransport.NewLeadHandler(readLeadsUseCase)
	authHandler := httptransport.NewAuthHandler(authService)
	importHandler := httptransport.NewImportHandler(importListingsUseCase)

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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(httptransport.AuthMiddleware(tokenService))

		r.Get("/api/me", authHandler.Me)
		leadHandler.RegisterRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(httptransport.AuthMiddleware(tokenService))
		r.Use(httptransport.RequireRole(user.RoleAdmin, user.RoleAnalyst))

		listingHandler.RegisterRoutes(r)
		r.Post("/api/imports/clean-listings", importHandler.ImportCleanListings)
		r.Get("/api/imports/{jobId}", importHandler.GetImportJob)
	})

	log.Println("server running on :8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}