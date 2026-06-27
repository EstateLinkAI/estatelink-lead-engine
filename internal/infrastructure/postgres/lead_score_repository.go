package postgres

import (
	"context"
	"encoding/json"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
	"github.com/jackc/pgx/v5/pgxpool"
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

	// Upsert on listing_id so re-importing/re-scoring a listing replaces its
	// score in place instead of accumulating duplicate rows.
	query := `
		INSERT INTO lead_scores (
			listing_id,
			score,
			grade,
			reasons
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (listing_id)
		DO UPDATE SET
			score = EXCLUDED.score,
			grade = EXCLUDED.grade,
			reasons = EXCLUDED.reasons,
			created_at = NOW()
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
