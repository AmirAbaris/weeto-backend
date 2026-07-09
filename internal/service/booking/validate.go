package booking

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	minNameLen  = 1
	maxNameLen  = 120
	minPhoneLen = 5
	maxPhoneLen = 32
	maxEmailLen = 254
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func validateCandidate(name, phone, email string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	phone = strings.TrimSpace(phone)
	email = strings.TrimSpace(strings.ToLower(email))

	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		return "", "", "", ErrInvalidName
	}
	if utf8.RuneCountInString(phone) < minPhoneLen || utf8.RuneCountInString(phone) > maxPhoneLen {
		return "", "", "", ErrInvalidPhone
	}
	if email == "" || utf8.RuneCountInString(email) > maxEmailLen || !emailPattern.MatchString(email) {
		return "", "", "", ErrInvalidEmail
	}

	return name, phone, email, nil
}
