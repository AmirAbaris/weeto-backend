package organization

import "errors"

var (
	ErrOrgNotFound      = errors.New("organization not found")
	ErrOrgAlreadyExists = errors.New("user already has an organization")
	ErrSlugTaken        = errors.New("slug already taken")
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidName      = errors.New("invalid organization name")
	ErrInvalidSlug      = errors.New("invalid slug")
	ErrInvalidPlan      = errors.New("invalid plan")
	ErrInvalidOwner = errors.New("invalid owner")
)
