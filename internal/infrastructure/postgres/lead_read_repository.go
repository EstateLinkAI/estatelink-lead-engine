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
	db                *pgxpool.Pool
	strategyScoreRepo *PropertyStrategyScoreRepository
}

func NewLeadReadRepository(db *pgxpool.Pool, strategyScoreRepo *PropertyStrategyScoreRepository) *LeadReadRepository {
	return &LeadReadRepository{db: db, strategyScoreRepo: strategyScoreRepo}
}

const leadFilterWhereClause = `
	($1 = '' OR LOWER(l.city) = LOWER($1))
	AND ($2 = '' OR LOWER(l.postcode_area) = LOWER($2))
	AND ($3 = '' OR LOWER(l.property_type) = LOWER($3))
	AND ($4 = '' OR LOWER(l.source_platform) = LOWER($4))
	AND ($5::int IS NULL OR ls.score >= $5)
`

func (r *LeadReadRepository) List(ctx context.Context, filters lead.ListFilters) (lead.ListResult, error) {
	filterArgs := []any{
		filters.City,
		filters.PostcodeArea,
		filters.PropertyType,
		filters.SourcePlatform,
		filters.MinScore,
	}

	total, err := r.count(ctx, filterArgs)
	if err != nil {
		return lead.ListResult{}, err
	}

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
		WHERE ` + leadFilterWhereClause + `
		ORDER BY ls.score DESC, ls.created_at DESC, ls.id DESC
		LIMIT $6 OFFSET $7;
	`

	rows, err := r.db.Query(
		ctx,
		query,
		append(filterArgs, filters.Limit, filters.Offset)...,
	)
	if err != nil {
		return lead.ListResult{}, err
	}
	defer rows.Close()

	leads := []lead.ReadModel{}

	for rows.Next() {
		item, err := scanLeadReadModel(rows)
		if err != nil {
			return lead.ListResult{}, err
		}

		leads = append(leads, item)
	}

	if err := rows.Err(); err != nil {
		return lead.ListResult{}, err
	}

	if err := r.attachStrategyScores(ctx, leads); err != nil {
		return lead.ListResult{}, err
	}

	return lead.ListResult{Items: leads, Total: total}, nil
}

func (r *LeadReadRepository) count(ctx context.Context, filterArgs []any) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM lead_scores ls
		INNER JOIN listings l ON l.id = ls.listing_id
		WHERE ` + leadFilterWhereClause + `;
	`

	var total int
	if err := r.db.QueryRow(ctx, query, filterArgs...).Scan(&total); err != nil {
		return 0, err
	}

	return total, nil
}

func (r *LeadReadRepository) attachStrategyScores(ctx context.Context, leads []lead.ReadModel) error {
	if r.strategyScoreRepo == nil || len(leads) == 0 {
		return nil
	}

	listingIDs := make([]int64, 0, len(leads))
	for _, item := range leads {
		listingIDs = append(listingIDs, item.ListingID)
	}

	grouped, err := r.strategyScoreRepo.ListByListingIDs(ctx, listingIDs)
	if err != nil {
		return err
	}

	for i := range leads {
		leads[i].StrategyScores = grouped[leads[i].ListingID]
	}

	return nil
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

	items := []lead.ReadModel{item}
	if err := r.attachStrategyScores(ctx, items); err != nil {
		return nil, err
	}

	return &items[0], nil
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
