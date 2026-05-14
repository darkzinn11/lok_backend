-- 000001_init_schema.down.sql

DROP INDEX IF EXISTS idx_visits_date;
DROP INDEX IF EXISTS idx_visits_status;
DROP INDEX IF EXISTS idx_visits_salesperson;

DROP TABLE IF EXISTS visit_photos;
DROP TABLE IF EXISTS visits;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS "uuid-ossp";
