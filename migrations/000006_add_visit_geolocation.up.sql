ALTER TABLE visits
    ADD COLUMN IF NOT EXISTS geo_latitude DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS geo_longitude DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS geo_accuracy_meters DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS geo_captured_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS geo_reverse_address TEXT;

CREATE INDEX IF NOT EXISTS idx_visits_geo_captured_at ON visits(geo_captured_at DESC);
