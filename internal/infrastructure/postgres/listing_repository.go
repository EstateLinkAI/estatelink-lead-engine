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

// Create upserts a listing. When the scraper supplies an external property
// ID, re-importing the same (source_platform, external_property_id) pair
// updates the existing row instead of inserting a duplicate; listings
// without an external ID (legacy/manual imports) always insert a new row.
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
			source_url,
			external_property_id
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (source_platform, external_property_id) WHERE external_property_id <> ''
		DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			price = EXCLUDED.price,
			city = EXCLUDED.city,
			postcode = EXCLUDED.postcode,
			postcode_area = EXCLUDED.postcode_area,
			property_type = EXCLUDED.property_type,
			bedrooms = EXCLUDED.bedrooms,
			bathrooms = EXCLUDED.bathrooms,
			rental_estimate = EXCLUDED.rental_estimate,
			market_price_estimate = EXCLUDED.market_price_estimate,
			days_on_market = EXCLUDED.days_on_market,
			source_url = EXCLUDED.source_url,
			updated_at = NOW()
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
		l.ExternalPropertyID,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)

	if err != nil {
		return listing.Listing{}, err
	}

	return l, nil
}
