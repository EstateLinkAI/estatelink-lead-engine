package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nath070707/estatelink-lead-engine/internal/application/ingestlisting"
	"github.com/nath070707/estatelink-lead-engine/internal/infrastructure/postgres"
	httptransport "github.com/nath070707/estatelink-lead-engine/internal/transport/http"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	ctx := context.Background()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	listingRepo := postgres.NewListingRepository(db)
	leadScoreRepo := postgres.NewLeadScoreRepository(db)

	ingestUseCase := ingestlisting.NewUseCase(listingRepo, leadScoreRepo)

	listingHandler := httptransport.NewListingHandler(ingestUseCase)

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	listingHandler.RegisterRoutes(r)

	log.Println("server running on :8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}