package auth

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const minPasswordLen = 8

var (
	ErrWeakPassword  = errors.New("password must be at least 8 characters")
	ErrInvalidEmail  = errors.New("invalid email")
)

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateCredentials(email, password string) (string, error) {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return "", ErrInvalidEmail
	}
	if utf8.RuneCountInString(password) < minPasswordLen {
		return "", ErrWeakPassword
	}
	return email, nil
}
