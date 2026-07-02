-- name: CreateOrganization :one
INSERT INTO organization (
    name,
    slug,
    logo_url,
    owner_id,
    plan
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetOrganizationByID :one
SELECT *
FROM organization
WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT *
FROM organization
WHERE slug = $1;

-- name: GetOrganizationsByOwner :one
SELECT *
FROM organization
WHERE owner_id = $1;

-- name: UpdateOrganization :one
UPDATE organization
SET
    name = $2,
    slug = $3,
    logo_url = $4,
    plan = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateOrganizationLogo :one
UPDATE organization
SET
    logo_url = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteOrganization :exec
DELETE FROM organization
WHERE id = $1;

-- name: OrganizationExistsBySlug :one
SELECT EXISTS (
    SELECT 1
    FROM organization
    WHERE slug = $1
);