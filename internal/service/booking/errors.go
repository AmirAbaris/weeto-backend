package booking

import "errors"

var (
	ErrOrgNotFound       = errors.New("organization not found")
	ErrTypeNotFound      = errors.New("interview type not found")
	ErrSlotNotFound      = errors.New("slot not found")
	ErrSlotUnavailable   = errors.New("slot is no longer available")
	ErrInvalidName       = errors.New("invalid candidate name")
	ErrInvalidPhone      = errors.New("invalid candidate phone")
	ErrInvalidEmail      = errors.New("invalid candidate email")
	ErrInvalidSlotID     = errors.New("invalid slot id")
)
