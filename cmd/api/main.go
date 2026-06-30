package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/auth"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/importlistings"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/logactivity"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/readleads"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/scorestrategies"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/config"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/infrastructure/postgres"
	httptransport "github.com/EstateLinkAI/estatelink-lead-engine/internal/transport/http"
	"github.com/EstateLinkAI/estatelink-lead-engine/migrations"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	config.LoadEnv()

	databaseURL := mustGetEnv("DATABASE_URL")
	jwtSecret := mustGetEnv("JWT_SECRET")

	accessTokenTTL := getDurationEnv("ACCESS_TOKEN_TTL", 24*time.Hour)
	refreshTokenTTL := getDurationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour)

	// Import safety limits - see internal/application/importlistings.UseCase
	// and internal/transport/http.ImportHandler. These exist to bound memory
	// usage for bulk imports after a 368k-row import crashed staging.
	maxImportRows := getIntEnv("MAX_IMPORT_ROWS", 5000)
	maxRequestBodyBytes := getIntEnv("MAX_REQUEST_BODY_BYTES", 25000000)
	importWorkers := getIntEnv("IMPORT_WORKERS", 4)

	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("failed to parse database url: %v", err)
	}

	// Bulk imports process listings concurrently (see importlistings.UseCase),
	// so the pool needs enough headroom for those workers plus normal API traffic.
	if poolConfig.MaxConns < 25 {
		poolConfig.MaxConns = 25
	}

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Run pending migrations on every boot so the schema the binary expects
	// can never silently drift from what's actually deployed (a missing
	// migration here previously made every row of a bulk import fail with
	// "column ... does not exist", with no way to tell from the API alone).
	if err := runMigrations(databaseURL); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Repositories
	listingRepo := postgres.NewListingRepository(db)
	leadScoreRepo := postgres.NewLeadScoreRepository(db)
	strategyScoreRepo := postgres.NewPropertyStrategyScoreRepository(db)
	leadReadRepo := postgres.NewLeadReadRepository(db, strategyScoreRepo)
	userRepo := postgres.NewUserRepository(db)
	rawListingRepo := postgres.NewRawListingRepository(db)
	importJobRepo := postgres.NewImportJobRepository(db)
	activityLogRepo := postgres.NewActivityLogRepository(db)

	// Use cases / services
	strategyScorer := scorestrategies.NewUseCase(strategyScoreRepo)

	ingestUseCase := ingestlisting.NewUseCase(
		listingRepo,
		leadScoreRepo,
		strategyScorer,
	)

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
		maxImportRows,
		importWorkers,
	)

	// Handlers
	listingHandler := httptransport.NewListingHandler(ingestUseCase, activityService)
	leadHandler := httptransport.NewLeadHandler(readLeadsUseCase)
	authHandler := httptransport.NewAuthHandler(authService, activityService)
	importHandler := httptransport.NewImportHandler(importListingsUseCase, int64(maxRequestBodyBytes))
	activityHandler := httptransport.NewActivityLogHandler(activityService)

	// Router
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: getCSVEnv("ALLOWED_ORIGINS", []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}),
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
		r.Get("/api/imports", importHandler.ListImportJobs)
		r.Get("/api/imports/{jobId}", importHandler.GetImportJob)
		r.Post("/api/imports/{jobId}/cancel", importHandler.CancelImportJob)
	})

	// Admin-only routes
	r.Group(func(r chi.Router) {
		r.Use(httptransport.AuthMiddleware(tokenService))
		r.Use(httptransport.RequireRole(user.RoleAdmin))

		r.Patch("/api/admin/users/{id}/role", authHandler.UpdateUserRole)

		activityHandler.RegisterRoutes(r)
	})

	port := getEnv("PORT", "8080")
	addr := ":" + port

	log.Printf("server running on %s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}

}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// runMigrations applies any pending goose migrations embedded in the
// migrations package. It uses its own short-lived database/sql connection
// (via the pgx stdlib driver) since goose operates on *sql.DB, separate from
// the pgxpool used for normal request handling.
func runMigrations(databaseURL string) error {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Up(sqlDB, ".")
}

func mustGetEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s environment variable is required", name)
	}

	return value

}

func getEnv(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value

}

func getCSVEnv(name string, fallback []string) []string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return fallback
	}

	return result

}

func getIntEnv(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("%s must be an integer, got %q: %v", name, value, err)
	}

	return parsed

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
