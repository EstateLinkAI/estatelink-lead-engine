package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/readleads"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/lead"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadReadRepository struct {
	db *pgxpool.Pool
}

func NewLeadReadRepository(db *pgxpool.Pool) *LeadReadRepository {
	return &LeadReadRepository{db: db}
}

func (r *LeadReadRepository) List(ctx context.Context, filters lead.ListFilters) ([]lead.ReadModel, error) {
	query := `
		SELECT
			ls.id,
			ls.listing_id,
			l.title,
			l.city,
			l.postcode,
			l.postcode_area,
			l.property_type,
			l.price,
			l.bedrooms,
			l.source_platform,
			ls.score,
			ls.grade,
			ls.reasons
		FROM lead_scores ls
		INNER JOIN listings l ON l.id = ls.listing_id
		WHERE
			($1 = '' OR LOWER(l.city) = LOWER($1))
			AND ($2 = '' OR LOWER(l.postcode_area) = LOWER($2))
			AND ($3 = '' OR LOWER(l.property_type) = LOWER($3))
			AND ($4 = '' OR LOWER(l.source_platform) = LOWER($4))
			AND ($5::int IS NULL OR ls.score >= $5)
		ORDER BY ls.score DESC, ls.created_at DESC
		LIMIT $6 OFFSET $7;
	`

	rows, err := r.db.Query(
		ctx,
		query,
		filters.City,
		filters.PostcodeArea,
		filters.PropertyType,
		filters.SourcePlatform,
		filters.MinScore,
		filters.Limit,
		filters.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leads := []lead.ReadModel{}

	for rows.Next() {
		item, err := scanLeadReadModel(rows)
		if err != nil {
			return nil, err
		}

		leads = append(leads, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return leads, nil
}

func (r *LeadReadRepository) GetByID(ctx context.Context, id string) (*lead.ReadModel, error) {
	query := `
		SELECT
			ls.id,
			ls.listing_id,
			l.title,
			l.city,
			l.postcode,
			l.postcode_area,
			l.property_type,
			l.price,
			l.bedrooms,
			l.source_platform,
			ls.score,
			ls.grade,
			ls.reasons
		FROM lead_scores ls
		INNER JOIN listings l ON l.id = ls.listing_id
		WHERE ls.id::text = $1
		LIMIT 1;
	`

	item, err := scanLeadReadModelRow(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, readleads.ErrLeadNotFound
	}

	if err != nil {
		return nil, err
	}

	return &item, nil
}

type leadScanner interface {
	Scan(dest ...any) error
}

func scanLeadReadModelRow(row leadScanner) (lead.ReadModel, error) {
	var item lead.ReadModel
	var reasonsJSON []byte

	err := row.Scan(
		&item.ID,
		&item.ListingID,
		&item.Title,
		&item.City,
		&item.Postcode,
		&item.PostcodeArea,
		&item.PropertyType,
		&item.Price,
		&item.Bedrooms,
		&item.SourcePlatform,
		&item.Score,
		&item.Grade,
		&reasonsJSON,
	)
	if err != nil {
		return lead.ReadModel{}, err
	}

	if err := json.Unmarshal(reasonsJSON, &item.Reasons); err != nil {
		return lead.ReadModel{}, err
	}

	return item, nil
}

func scanLeadReadModel(rows pgx.Rows) (lead.ReadModel, error) {
	return scanLeadReadModelRow(rows)
}
