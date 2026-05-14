DROP INDEX IF EXISTS idx_visits_geo_captured_at;

ALTER TABLE visits
    DROP COLUMN IF EXISTS geo_reverse_address,
    DROP COLUMN IF EXISTS geo_captured_at,
    DROP COLUMN IF EXISTS geo_accuracy_meters,
    DROP COLUMN IF EXISTS geo_longitude,
    DROP COLUMN IF EXISTS geo_latitude;
