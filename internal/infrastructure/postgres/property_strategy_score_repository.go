package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/strategy"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PropertyStrategyScoreRepository struct {
	db *pgxpool.Pool
}

func NewPropertyStrategyScoreRepository(db *pgxpool.Pool) *PropertyStrategyScoreRepository {
	return &PropertyStrategyScoreRepository{db: db}
}

func (r *PropertyStrategyScoreRepository) SaveMany(
	ctx context.Context,
	scores []strategy.StrategyScore,
) ([]strategy.StrategyScore, error) {
	if len(scores) == 0 {
		return []strategy.StrategyScore{}, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin strategy score transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO property_strategy_scores (
			listing_id,
			strategy,
			score,
			grade,
			reasons,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (listing_id, strategy)
		DO UPDATE SET
			score = EXCLUDED.score,
			grade = EXCLUDED.grade,
			reasons = EXCLUDED.reasons,
			created_at = EXCLUDED.created_at
		RETURNING id, created_at
	`

	savedScores := make([]strategy.StrategyScore, 0, len(scores))

	for _, score := range scores {
		reasonsJSON, err := json.Marshal(score.Reasons)
		if err != nil {
			return nil, fmt.Errorf("marshal strategy score reasons: %w", err)
		}

		savedScore := score

		err = tx.QueryRow(
			ctx,
			query,
			score.ListingID,
			string(score.Strategy),
			score.Score,
			score.Grade,
			reasonsJSON,
			score.CreatedAt,
		).Scan(
			&savedScore.ID,
			&savedScore.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"save strategy score for listing %d strategy %s: %w",
				score.ListingID,
				score.Strategy,
				err,
			)
		}

		savedScores = append(savedScores, savedScore)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit strategy score transaction: %w", err)
	}

	return savedScores, nil
}
