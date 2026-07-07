package sms

import (
	"errors"
	"strings"
	"unicode"
)

var ErrInvalidPhone = errors.New("invalid phone number")

// NormalizeMobile converts Iranian phone input to SMS.ir format: 10 digits starting with 9.
// Accepts 0912..., +98912..., 98912..., 912...
func NormalizeMobile(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	var digits strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	n := digits.String()
	switch {
	case len(n) == 10 && strings.HasPrefix(n, "9"):
		return n, nil
	case len(n) == 11 && strings.HasPrefix(n, "09"):
		return n[1:], nil
	case len(n) == 12 && strings.HasPrefix(n, "98"):
		return n[2:], nil
	default:
		return "", ErrInvalidPhone
	}
}
