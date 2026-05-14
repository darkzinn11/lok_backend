-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, role, branch_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY name;

-- name: UpdateUser :one
UPDATE users
SET name = $2, email = $3, password_hash = $4, role = $5, branch_id = $6, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
