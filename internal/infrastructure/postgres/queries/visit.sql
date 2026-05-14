-- name: CreateVisit :one
INSERT INTO visits (
    salesperson_id, status, date, client_name, client_cnpj, client_email,
    contact_phone, branch_phone, address, subject,
    conclusion, arrival_time, departure_time, km_start, km_end, observations
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: GetVisitByID :one
SELECT * FROM visits WHERE id = $1 LIMIT 1;

-- name: ListVisitsBySalesperson :many
SELECT * FROM visits WHERE salesperson_id = $1 ORDER BY date DESC;

-- name: ListVisitsByBranch :many
SELECT v.* FROM visits v
JOIN users u ON v.salesperson_id = u.id
WHERE u.branch_id = $1
ORDER BY v.date DESC;

-- name: ListAllVisits :many
SELECT * FROM visits ORDER BY date DESC;

-- name: UpdateVisit :one
UPDATE visits
SET status = $2, date = $3, client_name = $4, client_cnpj = $5, client_email = $6,
    contact_phone = $7, branch_phone = $8, address = $9,
    subject = $10, conclusion = $11, arrival_time = $12, departure_time = $13,
    km_start = $14, km_end = $15, observations = $16, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateVisitStatus :one
UPDATE visits
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: AddVisitPhoto :one
INSERT INTO visit_photos (visit_id, bucket_key)
VALUES ($1, $2)
RETURNING *;

-- name: ListVisitPhotos :many
SELECT * FROM visit_photos WHERE visit_id = $1;

-- name: DeleteVisit :exec
DELETE FROM visits WHERE id = $1;
