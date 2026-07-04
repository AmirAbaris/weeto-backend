-- name: InsertNotificationOutbox :one
INSERT INTO notification_outbox (
    organization_id,
    event_type,
    payload
)
VALUES ($1, $2, $3)
RETURNING *;
