-- name: CreateUser :one
INSERT INTO users (id, first_name, last_name, email, password)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email;

-- name: GetUserById :one
SELECT id, first_name, last_name, email, created_at, updated_at FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, password FROM users
WHERE email = $1 LIMIT 1;

-- name: ListUsers :many
SELECT id, first_name, last_name, email, created_at, updated_at FROM users
ORDER BY id
LIMIT $1
OFFSET $2;
