DROP INDEX IF EXISTS idx_branches_status;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_branch_id;

ALTER TABLE users DROP COLUMN IF EXISTS status;
ALTER TABLE users DROP COLUMN IF EXISTS phone;

DROP TABLE IF EXISTS branches;
