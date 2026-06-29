-- name: GetUserByEmail :one
SELECT id, email, created_at, updated_at
FROM users
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (email)
VALUES ($1)
RETURNING id, email, created_at, updated_at;