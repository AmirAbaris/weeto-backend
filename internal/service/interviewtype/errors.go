package interviewtype

import "errors"

var (
	ErrNotFound                = errors.New("interview type not found")
	ErrForbidden               = errors.New("forbidden")
	ErrSlugTaken               = errors.New("slug already taken")
	ErrInvalidTitle            = errors.New("invalid title")
	ErrInvalidSlug             = errors.New("invalid slug")
	ErrInvalidDuration         = errors.New("invalid duration")
	ErrInvalidBuffer           = errors.New("invalid buffer")
	ErrInvalidMeetingProvider  = errors.New("invalid meeting provider")
	ErrInvalidMeetingURL       = errors.New("invalid meeting url")
	ErrOrgRequired             = errors.New("organization required")
	ErrPlanLimitInterviewTypes = errors.New("free plan allows up to 3 interview types")
	ErrGoogleNotConnected      = errors.New("connect google to use google meet interview types")
)
