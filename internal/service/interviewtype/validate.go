package interviewtype

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/AmirAbaris/weeto-backend/internal/db"
)

const (
	minSlugLen       = 2
	maxSlugLen       = 48
	maxTitleLen      = 100
	minDurationMins  = 5
	maxDurationMins  = 480
	minBufferMins    = 0
	maxBufferMins    = 120
	freePlanMaxTypes = 3
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Input struct {
	Title           string
	Slug            string
	DurationMinutes int32
	BufferMinutes   int32
	MeetingProvider db.MeetingProvider
	MeetingURL      *string
}

func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func normalizeTitle(title string) string {
	return strings.TrimSpace(title)
}

func validateFields(in Input) (Input, error) {
	in.Title = normalizeTitle(in.Title)
	in.Slug = normalizeSlug(in.Slug)

	if in.Title == "" || utf8.RuneCountInString(in.Title) > maxTitleLen {
		return Input{}, ErrInvalidTitle
	}
	if utf8.RuneCountInString(in.Slug) < minSlugLen || utf8.RuneCountInString(in.Slug) > maxSlugLen || !slugPattern.MatchString(in.Slug) {
		return Input{}, ErrInvalidSlug
	}
	if in.DurationMinutes < minDurationMins || in.DurationMinutes > maxDurationMins {
		return Input{}, ErrInvalidDuration
	}
	if in.BufferMinutes < minBufferMins || in.BufferMinutes > maxBufferMins {
		return Input{}, ErrInvalidBuffer
	}
	if err := validateMeetingProvider(in.MeetingProvider, in.MeetingURL); err != nil {
		return Input{}, err
	}

	return in, nil
}

func validateMeetingProvider(provider db.MeetingProvider, meetingURL *string) error {
	switch provider {
	case db.MeetingProviderGoogleMeet:
		if meetingURL != nil && strings.TrimSpace(*meetingURL) != "" {
			return ErrInvalidMeetingURL
		}
	case db.MeetingProviderOnSite:
		if meetingURL == nil || strings.TrimSpace(*meetingURL) == "" {
			return ErrInvalidMeetingURL
		}
	default:
		return ErrInvalidMeetingProvider
	}
	return nil
}

func optionalMeetingURL(provider db.MeetingProvider, meetingURL *string) (string, bool) {
	switch provider {
	case db.MeetingProviderGoogleMeet:
		return "", false
	default:
		if meetingURL == nil {
			return "", false
		}
		trimmed := strings.TrimSpace(*meetingURL)
		return trimmed, trimmed != ""
	}
}
