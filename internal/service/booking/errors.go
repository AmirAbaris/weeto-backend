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
	ErrBookingNotFound   = errors.New("booking not found")
	ErrOrgRequired       = errors.New("organization required")
	ErrForbidden         = errors.New("forbidden")
	ErrGoogleNotConnected      = errors.New("connect google to use google meet interview types")
	ErrMeetLinkLimitReached    = errors.New("free plan google meet link limit reached for this month")
	ErrGoogleCalendarFailed    = errors.New("failed to create google calendar event")
	ErrTokenNotFound           = errors.New("booking token not found")
	ErrModifyCutoff            = errors.New("reschedule and cancel are disabled within the cutoff window before the interview")
	ErrBookingNotModifiable    = errors.New("booking cannot be modified")
	ErrSameSlot                = errors.New("cannot reschedule to the same slot")
)
