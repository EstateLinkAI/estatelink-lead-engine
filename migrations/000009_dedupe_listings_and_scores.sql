-- +goose Up

ALTER TABLE listings
ADD COLUMN external_property_id TEXT NOT NULL DEFAULT '';

-- Only enforce uniqueness when the scraper actually supplied a property ID,
-- so legacy/manual rows without one don't collide with each other.
CREATE UNIQUE INDEX IF NOT EXISTS listings_source_platform_external_property_id_unique
ON listings (source_platform, external_property_id)
WHERE external_property_id <> '';

-- Collapse any pre-existing duplicate lead_scores rows per listing before
-- adding the uniqueness constraint, keeping the most recent score.
DELETE FROM lead_scores ls
WHERE ls.id NOT IN (
    SELECT DISTINCT ON (listing_id) id
    FROM lead_scores
    ORDER BY listing_id, created_at DESC, id DESC
);

ALTER TABLE lead_scores
ADD CONSTRAINT lead_scores_listing_id_unique UNIQUE (listing_id);

-- +goose Down

ALTER TABLE lead_scores
DROP CONSTRAINT IF EXISTS lead_scores_listing_id_unique;

DROP INDEX IF EXISTS listings_source_platform_external_property_id_unique;

ALTER TABLE listings
DROP COLUMN IF EXISTS external_property_id;
