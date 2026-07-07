-- name: InsertNotificationOutbox :one
INSERT INTO notification_outbox (
    organization_id,
    event_type,
    payload
)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListPendingNotificationOutbox :many
SELECT id, organization_id, event_type, payload, status, retry_count, created_at, processed_at
FROM notification_outbox
WHERE status = 'pending'
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkNotificationOutboxSent :exec
UPDATE notification_outbox
SET status = 'sent', processed_at = NOW()
WHERE id = $1;

-- name: MarkNotificationOutboxFailed :exec
UPDATE notification_outbox
SET status = 'failed', retry_count = retry_count + 1, processed_at = NOW()
WHERE id = $1;
