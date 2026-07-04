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
