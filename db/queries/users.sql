-- name: GetUserByEmail :one
SELECT id, email, password_hash, last_login_at, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, last_login_at, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, password_hash, last_login_at, created_at, updated_at;

-- name: TouchLastLogin :exec
UPDATE users
SET last_login_at = now(), updated_at = now()
WHERE id = $1;

-- name: IsGoogleConnected :one
SELECT (google_connected_at IS NOT NULL)::bool AS connected
FROM users
WHERE id = $1;

-- name: GetUserGoogleCredentials :one
SELECT google_id, google_refresh_token, google_connected_at
FROM users
WHERE id = $1;

-- name: SetUserGoogleConnection :exec
UPDATE users
SET
    google_id = $2,
    google_refresh_token = $3,
    google_connected_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: ClearUserGoogleConnection :exec
UPDATE users
SET
    google_id = NULL,
    google_refresh_token = NULL,
    google_connected_at = NULL,
    updated_at = NOW()
WHERE id = $1;