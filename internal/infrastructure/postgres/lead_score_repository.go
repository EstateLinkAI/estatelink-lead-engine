package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nath070707/estatelink-lead-engine/internal/domain/lead"
)

type LeadScoreRepository struct {
	db *pgxpool.Pool
}

func NewLeadScoreRepository(db *pgxpool.Pool) *LeadScoreRepository {
	return &LeadScoreRepository{db: db}
}

func (r *LeadScoreRepository) Create(ctx context.Context, score lead.Score) (lead.Score, error) {
	reasonsJSON, err := json.Marshal(score.Reasons)
	if err != nil {
		return lead.Score{}, err
	}

	query := `
		INSERT INTO lead_scores (
			listing_id,
			score,
			grade,
			reasons
		)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at;
	`

	err = r.db.QueryRow(
		ctx,
		query,
		score.ListingID,
		score.Value,
		string(score.Grade),
		reasonsJSON,
	).Scan(&score.CreatedAt)

	if err != nil {
		return lead.Score{}, err
	}

	return score, nil
}