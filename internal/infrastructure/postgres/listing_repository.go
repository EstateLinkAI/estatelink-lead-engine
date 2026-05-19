package postgres

import (
	"context"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/listing"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ListingRepository struct {
	db *pgxpool.Pool
}

func NewListingRepository(db *pgxpool.Pool) *ListingRepository {
	return &ListingRepository{db: db}
}

func (r *ListingRepository) Create(ctx context.Context, l listing.Listing) (listing.Listing, error) {
	query := `
		INSERT INTO listings (
			title,
			description,
			price,
			city,
			postcode,
			postcode_area,
			property_type,
			bedrooms,
			bathrooms,
			rental_estimate,
			market_price_estimate,
			days_on_market,
			source_platform,
			source_url
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14
		)
		RETURNING id, created_at, updated_at;
	`

	err := r.db.QueryRow(
		ctx,
		query,
		l.Title,
		l.Description,
		l.Price,
		l.City,
		l.Postcode,
		l.PostcodeArea,
		l.PropertyType,
		l.Bedrooms,
		l.Bathrooms,
		l.RentalEstimate,
		l.MarketPriceEstimate,
		l.DaysOnMarket,
		l.SourcePlatform,
		l.SourceURL,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)

	if err != nil {
		return listing.Listing{}, err
	}

	return l, nil
}
