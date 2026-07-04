-- name: GetSlotForUpdate :one
SELECT *
FROM slots
WHERE id = $1
  AND organization_id = $2
  AND interview_type_id = $3
  AND booked = FALSE
FOR UPDATE;

-- name: MarkSlotBooked :execrows
UPDATE slots
SET booked = TRUE
WHERE id = $1
  AND booked = FALSE;

-- name: InsertBooking :one
INSERT INTO booking (
    organization_id,
    interview_type_id,
    slot_id,
    candidate_name,
    candidate_phone,
    candidate_email,
    reschedule_token,
    cancel_token
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetBookingByID :one
SELECT *
FROM booking
WHERE id = $1;

-- name: CountBookingsBySlot :one
SELECT COUNT(*)::int
FROM booking
WHERE slot_id = $1;

-- name: ListNotificationOutboxByOrg :many
SELECT *
FROM notification_outbox
WHERE organization_id = $1
ORDER BY created_at;

-- name: ListScheduledBookingsByOrg :many
SELECT
    b.*,
    s.start_at,
    s.end_at,
    it.title AS interview_type_title
FROM booking b
JOIN slots s ON s.id = b.slot_id
JOIN interview_type it ON it.id = b.interview_type_id
WHERE b.organization_id = $1
  AND b.status = 'scheduled'
  AND s.start_at >= $2
ORDER BY s.start_at;

-- name: GetScheduledBookingForUpdate :one
SELECT b.*
FROM booking b
WHERE b.id = $1
  AND b.organization_id = $2
  AND b.status = 'scheduled'
FOR UPDATE;

-- name: CancelBooking :one
UPDATE booking
SET status = 'cancelled', updated_at = NOW()
WHERE id = $1
  AND organization_id = $2
  AND status = 'scheduled'
RETURNING *;
