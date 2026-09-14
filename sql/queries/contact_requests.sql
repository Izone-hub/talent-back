-- name: CreateContactRequest :one
INSERT INTO contact_requests (
    first_name, last_name, email, company, budget_range, project_details
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListContactRequests :many
SELECT * FROM contact_requests
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountContactRequests :one
SELECT COUNT(*) FROM contact_requests;

-- name: GetContactRequest :one
SELECT * FROM contact_requests
WHERE id = $1;

-- name: ListContactRequestsByEmail :many
SELECT * FROM contact_requests
WHERE LOWER(email) = LOWER($1)
ORDER BY created_at DESC;

-- name: UpdateContactRequestStatus :one
UPDATE contact_requests
SET status = $2
WHERE id = $1
RETURNING *;

-- name: DeleteContactRequest :exec
DELETE FROM contact_requests
WHERE id = $1;
