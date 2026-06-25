-- +goose Up

CREATE TABLE property_images (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    source_url TEXT NOT NULL,
    position INT NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_property_images_listing_id
ON property_images(listing_id);

CREATE UNIQUE INDEX idx_property_images_listing_source_url
ON property_images(listing_id, source_url);

CREATE UNIQUE INDEX idx_property_images_one_primary_per_listing
ON property_images(listing_id)
WHERE is_primary = true;


CREATE TABLE property_strategy_scores (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    strategy TEXT NOT NULL,
    score INT NOT NULL CHECK (score >= 0 AND score <= 100),
    grade TEXT NOT NULL,
    reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_property_strategy_scores_listing_id
ON property_strategy_scores(listing_id);

CREATE INDEX idx_property_strategy_scores_strategy
ON property_strategy_scores(strategy);

CREATE INDEX idx_property_strategy_scores_score
ON property_strategy_scores(score DESC);

CREATE UNIQUE INDEX idx_property_strategy_scores_listing_strategy
ON property_strategy_scores(listing_id, strategy);

-- +goose Down

DROP TABLE IF EXISTS property_strategy_scores;
DROP TABLE IF EXISTS property_images;