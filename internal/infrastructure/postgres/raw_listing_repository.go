package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/rawlisting"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RawListingRepository struct {
	db *pgxpool.Pool
}

func NewRawListingRepository(db *pgxpool.Pool) *RawListingRepository {
	return &RawListingRepository{
		db: db,
	}
}

func (r *RawListingRepository) Save(ctx context.Context, listing rawlisting.RawListing) (string, error) {
	payload, err := json.Marshal(listing.RawPayload)
	if err != nil {
		return "", fmt.Errorf("marshal raw payload: %w", err)
	}

	const query = `
		INSERT INTO raw_listings (
			source,
			external_property_id,
			raw_payload,
			scraped_at,
			processing_status,
			updated_at
		)
		VALUES ($1, $2, $3::jsonb, $4, $5, NOW())
		ON CONFLICT (source, external_property_id)
		DO UPDATE SET
			raw_payload = EXCLUDED.raw_payload,
			scraped_at = EXCLUDED.scraped_at,
			processing_status = 'pending',
			error_message = NULL,
			updated_at = NOW()
		RETURNING id::text;
	`

	var id string

	err = r.db.QueryRow(
		ctx,
		query,
		listing.Source,
		listing.ExternalPropertyID,
		string(payload),
		listing.ScrapedAt,
		listing.ProcessingStatus,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("save raw listing query: %w", err)
	}

	return id, nil
}

func (r *RawListingRepository) MarkProcessed(ctx context.Context, id string) error {
	const query = `
		UPDATE raw_listings
		SET
			processing_status = 'processed',
			processed_at = NOW(),
			error_message = NULL,
			updated_at = NOW()
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark raw listing processed: %w", err)
	}

	return nil
}

func (r *RawListingRepository) MarkFailed(ctx context.Context, id string, reason string) error {
	const query = `
		UPDATE raw_listings
		SET
			processing_status = 'failed',
			error_message = $2,
			updated_at = NOW()
		WHERE id = $1;
	`

	_, err := r.db.Exec(ctx, query, id, reason)
	if err != nil {
		return fmt.Errorf("mark raw listing failed: %w", err)
	}

	return nil
}