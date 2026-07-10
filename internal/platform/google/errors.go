package google

import "errors"

var (
	ErrNotConnected = errors.New("google account not connected")
	ErrCalendarAPI  = errors.New("google calendar api error")
)
