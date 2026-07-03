package availability

import "errors"

var (
	ErrNotFound              = errors.New("availability not configured")
	ErrForbidden             = errors.New("forbidden")
	ErrOrgRequired           = errors.New("organization required")
	ErrInvalidTimezone       = errors.New("invalid timezone")
	ErrInvalidMaxPerDay      = errors.New("invalid max_interviews_per_day")
	ErrInvalidDayOfWeek      = errors.New("invalid day_of_week")
	ErrInvalidTimeRange      = errors.New("invalid time range")
	ErrOverlappingHours      = errors.New("overlapping working hours")
	ErrBreakOutsideHours     = errors.New("break outside working hours")
	ErrOverlappingTimeOff    = errors.New("overlapping time off blocks")
	ErrInvalidTimeOffRange   = errors.New("invalid time off range")
	ErrInvalidTimeOffTime    = errors.New("invalid time off timestamp")
)
