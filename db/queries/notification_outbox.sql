-- name: InsertNotificationOutbox :one
INSERT INTO notification_outbox (
    organization_id,
    event_type,
    payload
)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ClaimPendingNotifications :many
SELECT *
FROM notification_outbox
WHERE status = 'pending'
  AND event_type = 'booking_created'
  AND payload->>'recipient' = 'candidate'
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: MarkNotificationSent :one
UPDATE notification_outbox
SET status = 'sent',
    processed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkNotificationFailed :one
UPDATE notification_outbox
SET retry_count = retry_count + 1,
    status = CASE
        WHEN retry_count + 1 >= $2 THEN 'failed'::notification_status
        ELSE 'pending'::notification_status
    END,
    processed_at = CASE
        WHEN retry_count + 1 >= $2 THEN NOW()
        ELSE processed_at
    END
WHERE id = $1
RETURNING *;
