package slot

import "errors"

var (
	ErrAvailabilityNotConfigured = errors.New("availability not configured")
	ErrInvalidTimezone           = errors.New("invalid timezone")
)
