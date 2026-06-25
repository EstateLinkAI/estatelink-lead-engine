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

func (r *PropertyStrategyScoreRepository) ListByListingIDs(
	ctx context.Context,
	listingIDs []int64,
) (map[int64][]strategy.StrategyScore, error) {
	result := make(map[int64][]strategy.StrategyScore)

	if len(listingIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT id, listing_id, strategy, score, grade, reasons, created_at
		FROM property_strategy_scores
		WHERE listing_id = ANY($1)
		ORDER BY listing_id, score DESC
	`

	rows, err := r.db.Query(ctx, query, listingIDs)
	if err != nil {
		return nil, fmt.Errorf("list strategy scores: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item strategy.StrategyScore
		var strategyName string
		var reasonsJSON []byte

		if err := rows.Scan(
			&item.ID,
			&item.ListingID,
			&strategyName,
			&item.Score,
			&item.Grade,
			&reasonsJSON,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan strategy score: %w", err)
		}

		item.Strategy = strategy.Strategy(strategyName)

		if err := json.Unmarshal(reasonsJSON, &item.Reasons); err != nil {
			return nil, fmt.Errorf("unmarshal strategy score reasons: %w", err)
		}

		result[item.ListingID] = append(result[item.ListingID], item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate strategy scores: %w", err)
	}

	return result, nil
}

func (r *PropertyStrategyScoreRepository) ListByListingID(
	ctx context.Context,
	listingID int64,
) ([]strategy.StrategyScore, error) {
	grouped, err := r.ListByListingIDs(ctx, []int64{listingID})
	if err != nil {
		return nil, err
	}

	return grouped[listingID], nil
}
