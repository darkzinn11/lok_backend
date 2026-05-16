-- 000011_add_performance_indices.up.sql
CREATE INDEX IF NOT EXISTS idx_users_branch ON users(branch_id);
CREATE INDEX IF NOT EXISTS idx_visits_client_cnpj ON visits(client_cnpj);
CREATE INDEX IF NOT EXISTS idx_visits_client_name ON visits(client_name);
CREATE INDEX IF NOT EXISTS idx_visits_perf_agg ON visits(salesperson_id, date, status);
CREATE INDEX IF NOT EXISTS idx_visits_date_status ON visits(date, status);
