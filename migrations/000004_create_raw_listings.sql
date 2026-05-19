-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS raw_listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    external_property_id TEXT NOT NULL,
    raw_payload JSONB NOT NULL,
    scraped_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    processing_status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT raw_listings_source_external_property_id_unique
        UNIQUE (source, external_property_id)
);

CREATE INDEX IF NOT EXISTS idx_raw_listings_source
    ON raw_listings(source);

CREATE INDEX IF NOT EXISTS idx_raw_listings_processing_status
    ON raw_listings(processing_status);

CREATE INDEX IF NOT EXISTS idx_raw_listings_scraped_at
    ON raw_listings(scraped_at);

-- +goose Down
DROP TABLE IF EXISTS raw_listings;