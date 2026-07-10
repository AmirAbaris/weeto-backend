package google

import "errors"

var (
	ErrGoogleNotConfigured = errors.New("google integration not configured")
	ErrInvalidOAuthState   = errors.New("invalid oauth state")
)
