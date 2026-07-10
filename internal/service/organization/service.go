package organization

import (
	"context"
	"errors"
	"strings"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	q   *db.Queries
	cfg *config.Config
}

func NewService(q *db.Queries, cfg *config.Config) *Service {
	return &Service{q: q, cfg: cfg}
}

func (s *Service) CreateOrg(ctx context.Context, ownerID pgtype.UUID, name, slug string, logoURL *string) (db.Organization, error) {
	if !ownerID.Valid {
		return db.Organization{}, ErrInvalidOwner
	}

	name, slug, err := validateOrgFields(name, slug)
	if err != nil {
		return db.Organization{}, err
	}

	if _, err := s.q.GetOrganizationsByOwner(ctx, ownerID); err == nil {
		return db.Organization{}, ErrOrgAlreadyExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.Organization{}, err
	}

	if err := s.ensureSlugAvailable(ctx, slug); err != nil {
		return db.Organization{}, err
	}

	org, err := s.q.CreateOrganization(ctx, db.CreateOrganizationParams{
		Name:    name,
		Slug:    slug,
		LogoUrl: optionalText(logoURL),
		OwnerID: ownerID,
		Plan:    db.PlanTypeFree,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.Organization{}, mapUniqueViolation(err)
		}
		return db.Organization{}, err
	}

	return org, nil
}

func (s *Service) GetByID(ctx context.Context, id, ownerID pgtype.UUID) (db.Organization, error) {
	return s.getOrgForOwner(ctx, id, ownerID)
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (db.Organization, error) {
	slug = normalizeSlug(slug)
	if slug == "" {
		return db.Organization{}, ErrInvalidSlug
	}

	org, err := s.q.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Organization{}, ErrOrgNotFound
		}
		return db.Organization{}, err
	}

	return org, nil
}

func (s *Service) GetByOwner(ctx context.Context, ownerID pgtype.UUID) (db.Organization, error) {
	if !ownerID.Valid {
		return db.Organization{}, ErrInvalidOwner
	}

	org, err := s.q.GetOrganizationsByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Organization{}, ErrOrgNotFound
		}
		return db.Organization{}, err
	}

	return org, nil
}

func (s *Service) UpdateOrg(ctx context.Context, id, ownerID pgtype.UUID, name, slug string, logoURL *string) (db.Organization, error) {
	org, err := s.getOrgForOwner(ctx, id, ownerID)
	if err != nil {
		return db.Organization{}, err
	}

	name, slug, err = validateOrgFields(name, slug)
	if err != nil {
		return db.Organization{}, err
	}

	if slug != org.Slug {
		if err := s.ensureSlugAvailable(ctx, slug); err != nil {
			return db.Organization{}, err
		}
	}

	logo := org.LogoUrl
	if logoURL != nil {
		logo = optionalText(logoURL)
	}

	updated, err := s.q.UpdateOrganization(ctx, db.UpdateOrganizationParams{
		ID:      id,
		Name:    name,
		Slug:    slug,
		LogoUrl: logo,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.Organization{}, mapUniqueViolation(err)
		}
		return db.Organization{}, err
	}

	return updated, nil
}

func (s *Service) UpdatePlan(ctx context.Context, id pgtype.UUID, newPlan db.PlanType) (db.Organization, error) {
	if !id.Valid {
		return db.Organization{}, ErrInvalidOwner
	}
	if err := validatePlan(newPlan); err != nil {
		return db.Organization{}, err
	}
	if newPlan == "" {
		return db.Organization{}, ErrInvalidPlan
	}

	updated, err := s.q.UpdateOrganizationPlan(ctx, db.UpdateOrganizationPlanParams{
		ID:   id,
		Plan: newPlan,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Organization{}, ErrOrgNotFound
		}
		return db.Organization{}, err
	}

	return updated, nil
}

func (s *Service) UpdateLogo(ctx context.Context, id, ownerID pgtype.UUID, logoURL string) (db.Organization, error) {
	if _, err := s.getOrgForOwner(ctx, id, ownerID); err != nil {
		return db.Organization{}, err
	}

	logo := optionalText(&logoURL)

	updated, err := s.q.UpdateOrganizationLogo(ctx, db.UpdateOrganizationLogoParams{
		ID:      id,
		LogoUrl: logo,
	})
	if err != nil {
		return db.Organization{}, err
	}

	return updated, nil
}

func (s *Service) DeleteOrg(ctx context.Context, id, ownerID pgtype.UUID) error {
	if _, err := s.getOrgForOwner(ctx, id, ownerID); err != nil {
		return err
	}

	return s.q.DeleteOrganization(ctx, id)
}

func (s *Service) getOrgForOwner(ctx context.Context, id, ownerID pgtype.UUID) (db.Organization, error) {
	org, err := s.q.GetOrganizationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Organization{}, ErrOrgNotFound
		}
		return db.Organization{}, err
	}

	if err := ensureOwner(org, ownerID); err != nil {
		return db.Organization{}, err
	}

	return org, nil
}

func (s *Service) ensureSlugAvailable(ctx context.Context, slug string) error {
	exists, err := s.q.OrganizationExistsBySlug(ctx, slug)
	if err != nil {
		return err
	}
	if exists {
		return ErrSlugTaken
	}
	return nil
}

func ensureOwner(org db.Organization, ownerID pgtype.UUID) error {
	if !ownerID.Valid || !org.OwnerID.Valid || org.OwnerID.Bytes != ownerID.Bytes {
		return ErrForbidden
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func mapUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if strings.Contains(pgErr.ConstraintName, "owner") {
		return ErrOrgAlreadyExists
	}
	return ErrSlugTaken
}
