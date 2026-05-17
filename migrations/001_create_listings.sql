-- +goose Up
CREATE TABLE IF NOT EXISTS listings (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price INTEGER NOT NULL,
    city TEXT NOT NULL,
    postcode TEXT NOT NULL,
    postcode_area TEXT NOT NULL,
    property_type TEXT NOT NULL,
    bedrooms INTEGER NOT NULL DEFAULT 0,
    bathrooms INTEGER NOT NULL DEFAULT 0,
    rental_estimate INTEGER NOT NULL DEFAULT 0,
    market_price_estimate INTEGER NOT NULL DEFAULT 0,
    days_on_market INTEGER NOT NULL DEFAULT 0,
    source_platform TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS listings;