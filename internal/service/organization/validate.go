package organization

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	minSlugLen = 2
	maxSlugLen = 48
	maxNameLen = 100
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func validateOrgFields(name, slug string) (string, string, error) {
	name = normalizeName(name)
	slug = normalizeSlug(slug)

	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		return "", "", ErrInvalidName
	}
	if utf8.RuneCountInString(slug) < minSlugLen || utf8.RuneCountInString(slug) > maxSlugLen || !slugPattern.MatchString(slug) {
		return "", "", ErrInvalidSlug
	}
	return name, slug, nil
}

func validatePlan(plan db.PlanType) error {
	switch plan {
	case db.PlanTypeFree, db.PlanTypePro, db.PlanTypeBusiness, "":
		return nil
	default:
		return ErrInvalidPlan
	}
}

func optionalText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: trimmed, Valid: true}
}
