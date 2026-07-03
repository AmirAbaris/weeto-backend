-- name: DeleteUnbookedSlotsByOrgInWindow :exec
DELETE FROM slots
WHERE organization_id = $1
  AND booked = FALSE
  AND start_at >= $2
  AND start_at < $3;

-- name: DeleteUnbookedSlotsByTypeInWindow :exec
DELETE FROM slots
WHERE interview_type_id = $1
  AND booked = FALSE
  AND start_at >= $2
  AND start_at < $3;

-- name: InsertSlot :one
INSERT INTO slots (
    organization_id,
    interview_type_id,
    start_at,
    end_at
)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: BulkInsertSlots :copyfrom
INSERT INTO slots (
    organization_id,
    interview_type_id,
    start_at,
    end_at
) VALUES (
    $1, $2, $3, $4
);

-- name: CountSlotsByOrgOnLocalDay :one
-- $2 = local day start (UTC), $3 = local day end (UTC) — compute in Go from org timezone
SELECT COUNT(*)::int AS slot_count
FROM slots
WHERE organization_id = $1
  AND start_at >= $2
  AND start_at < $3;

-- name: ListAvailableSlotsByType :many
SELECT *
FROM slots
WHERE organization_id = $1
  AND interview_type_id = $2
  AND booked = FALSE
  AND start_at >= $3
  AND start_at < $4
ORDER BY start_at;

-- name: ListSlotsByTypeInWindow :many
SELECT *
FROM slots
WHERE interview_type_id = $1
  AND start_at >= $2
  AND start_at < $3
ORDER BY start_at;

-- name: SetSlotBooked :exec
UPDATE slots
SET booked = $2
WHERE id = $1;