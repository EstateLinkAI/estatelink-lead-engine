-- +goose Up
CREATE TABLE IF NOT EXISTS lead_scores (
    id BIGSERIAL PRIMARY KEY,
    listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    score INTEGER NOT NULL,
    grade TEXT NOT NULL,
    reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lead_scores_listing_id
ON lead_scores(listing_id);

CREATE INDEX IF NOT EXISTS idx_lead_scores_score
ON lead_scores(score);

-- +goose Down
DROP TABLE IF EXISTS lead_scores;