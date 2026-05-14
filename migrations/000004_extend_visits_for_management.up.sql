ALTER TABLE visits
    ADD COLUMN IF NOT EXISTS manager_observation TEXT;

ALTER TABLE visit_photos
    ADD COLUMN IF NOT EXISTS public_url TEXT;

ALTER TABLE visit_photos
    ADD COLUMN IF NOT EXISTS file_name VARCHAR(255);

ALTER TABLE visit_photos
    ADD COLUMN IF NOT EXISTS file_size BIGINT NOT NULL DEFAULT 0;

ALTER TABLE visit_photos
    ADD COLUMN IF NOT EXISTS photo_type VARCHAR(20) NOT NULL DEFAULT 'outra';

CREATE INDEX IF NOT EXISTS idx_visits_manager_observation ON visits(created_at DESC);
