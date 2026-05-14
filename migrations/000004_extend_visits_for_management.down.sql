ALTER TABLE visit_photos
    DROP COLUMN IF EXISTS photo_type;

ALTER TABLE visit_photos
    DROP COLUMN IF EXISTS file_size;

ALTER TABLE visit_photos
    DROP COLUMN IF EXISTS file_name;

ALTER TABLE visit_photos
    DROP COLUMN IF EXISTS public_url;

ALTER TABLE visits
    DROP COLUMN IF EXISTS manager_observation;
