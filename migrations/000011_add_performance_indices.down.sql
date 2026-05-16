-- 000011_add_performance_indices.down.sql
DROP INDEX IF EXISTS idx_users_branch;
DROP INDEX IF EXISTS idx_visits_client_cnpj;
DROP INDEX IF EXISTS idx_visits_client_name;
DROP INDEX IF EXISTS idx_visits_perf_agg;
DROP INDEX IF EXISTS idx_visits_date_status;
