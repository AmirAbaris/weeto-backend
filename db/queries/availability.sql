-- name: UpsertAvailabilitySettings :exec
INSERT INTO availability_settings (
    organization_id,
    timezone,
    max_interviews_per_day
)
VALUES ($1, $2, $3)
ON CONFLICT (organization_id) DO UPDATE
SET
    timezone = EXCLUDED.timezone,
    max_interviews_per_day = EXCLUDED.max_interviews_per_day,
    updated_at = NOW();

-- name: DeleteWorkingHoursByOrg :exec
DELETE FROM availability_working_hours
WHERE organization_id = $1;

-- name: InsertWorkingHour :exec
INSERT INTO availability_working_hours (
    organization_id,
    day_of_week,
    start_time,
    end_time
)
VALUES ($1, $2, $3, $4);

-- name: DeleteBreaksByOrg :exec
DELETE FROM availability_breaks
WHERE organization_id = $1;

-- name: InsertBreak :exec
INSERT INTO availability_breaks (
    organization_id,
    day_of_week,
    start_time,
    end_time
)
VALUES ($1, $2, $3, $4);

-- name: DeleteTimeOffByOrg :exec
DELETE FROM availability_time_off
WHERE organization_id = $1;

-- name: InsertTimeOff :one
INSERT INTO availability_time_off (
    organization_id,
    start_at,
    end_at
)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAvailabilitySettingsByOrg :one
SELECT *
FROM availability_settings
WHERE organization_id = $1;

-- name: ListWorkingHoursByOrg :many
SELECT *
FROM availability_working_hours
WHERE organization_id = $1
ORDER BY day_of_week, start_time;

-- name: ListBreaksByOrg :many
SELECT *
FROM availability_breaks
WHERE organization_id = $1
ORDER BY day_of_week, start_time;

-- name: ListTimeOffByOrg :many
SELECT *
FROM availability_time_off
WHERE organization_id = $1
ORDER BY start_at, end_at;

-- name: GetAvailabilityByOrg :one
SELECT
    s.organization_id,
    s.timezone,
    s.max_interviews_per_day,
    s.updated_at,
    COALESCE((
        SELECT json_agg(
            json_build_object(
                'id', wh.id,
                'day_of_week', wh.day_of_week,
                'start_time', wh.start_time,
                'end_time', wh.end_time,
                'created_at', wh.created_at
            )
            ORDER BY wh.day_of_week, wh.start_time
        )
        FROM availability_working_hours wh
        WHERE wh.organization_id = s.organization_id
    ), '[]'::json) AS working_hours,
    COALESCE((
        SELECT json_agg(
            json_build_object(
                'id', b.id,
                'day_of_week', b.day_of_week,
                'start_time', b.start_time,
                'end_time', b.end_time
            )
            ORDER BY b.day_of_week, b.start_time
        )
        FROM availability_breaks b
        WHERE b.organization_id = s.organization_id
    ), '[]'::json) AS breaks,
    COALESCE((
        SELECT json_agg(
            json_build_object(
                'id', t.id,
                'start_at', t.start_at,
                'end_at', t.end_at
            )
            ORDER BY t.start_at, t.end_at
        )
        FROM availability_time_off t
        WHERE t.organization_id = s.organization_id
    ), '[]'::json) AS time_off
FROM availability_settings s
WHERE s.organization_id = $1;