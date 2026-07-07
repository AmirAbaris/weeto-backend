-- name: InsertNotificationOutbox :one
INSERT INTO notification_outbox (
    organization_id,
    event_type,
    payload,
    scheduled_at
)
VALUES ($1, $2, $3, COALESCE(sqlc.narg('scheduled_at'), NOW()))
RETURNING *;

-- name: ClaimPendingNotifications :many
SELECT *
FROM notification_outbox
WHERE status = 'pending'
  AND scheduled_at <= NOW()
  AND (
    (event_type = 'booking_created' AND payload->>'recipient' IN ('candidate', 'recruiter'))
    OR (event_type = 'booking_rescheduled' AND payload->>'recipient' = 'candidate')
    OR (event_type = 'booking_cancelled' AND payload->>'recipient' IN ('candidate', 'recruiter'))
    OR (event_type = 'reminder_24h' AND payload->>'recipient' = 'candidate')
  )
ORDER BY scheduled_at, created_at
LIMIT $1
FOR UPDATE SKIP LOCKED;

-- name: CancelPendingRemindersByBookingID :execrows
UPDATE notification_outbox
SET status = 'cancelled',
    processed_at = NOW()
WHERE status = 'pending'
  AND event_type = 'reminder_24h'
  AND payload->>'booking_id' = sqlc.arg(booking_id)::text;

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
