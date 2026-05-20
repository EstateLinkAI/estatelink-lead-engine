-- +goose Up
CREATE TABLE IF NOT EXISTS import_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL DEFAULT 'queued',
    total_count INT NOT NULL DEFAULT 0,
    processed_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE raw_listings
ADD COLUMN IF NOT EXISTS import_job_id UUID REFERENCES import_jobs(id);

CREATE INDEX IF NOT EXISTS idx_import_jobs_status
    ON import_jobs(status);

CREATE INDEX IF NOT EXISTS idx_raw_listings_import_job_id
    ON raw_listings(import_job_id);

-- +goose Down
ALTER TABLE raw_listings
DROP COLUMN IF EXISTS import_job_id;

DROP TABLE IF EXISTS import_jobs;