-- name: CreateInterviewType :one
INSERT INTO interview_type (
    organization_id,
    title,
    slug,
    duration_minutes,
    buffer_minutes,
    meeting_provider,
    meeting_url
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetInterviewTypeByID :one
SELECT *
FROM interview_type
WHERE id = $1;

-- name: ListInterviewTypesByOrg :many
SELECT *
FROM interview_type
WHERE organization_id = $1
ORDER BY created_at;

-- name: UpdateInterviewType :one
UPDATE interview_type
SET
    title = $2,
    slug = $3,
    duration_minutes = $4,
    buffer_minutes = $5,
    meeting_provider = $6,
    meeting_url = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CountInterviewTypesByOrg :one
SELECT COUNT(*)::int
FROM interview_type
WHERE organization_id = $1;

-- name: InterviewTypeExistsBySlug :one
SELECT EXISTS (
    SELECT 1
    FROM interview_type
    WHERE organization_id = $1 AND slug = $2
);

-- name: GetInterviewTypeByOrgAndSlug :one
SELECT *
FROM interview_type
WHERE organization_id = $1 AND slug = $2;
