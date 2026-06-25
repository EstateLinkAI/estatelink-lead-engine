-- +goose Up

ALTER TABLE listings
ADD COLUMN estimated_monthly_rent NUMERIC NULL,
ADD COLUMN estimated_monthly_rent_source TEXT NULL,
ADD COLUMN estimated_monthly_rent_confidence INT NULL;

-- +goose Down

ALTER TABLE listings
DROP COLUMN IF EXISTS estimated_monthly_rent_confidence,
DROP COLUMN IF EXISTS estimated_monthly_rent_source,
DROP COLUMN IF EXISTS estimated_monthly_rent;