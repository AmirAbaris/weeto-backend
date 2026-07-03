package interviewtype

import (
	"context"
	"errors"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	slotsvc "github.com/AmirAbaris/weeto-backend/internal/service/slot"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	q       *db.Queries
	orgSvc  *orgsvc.Service
	slotSvc *slotsvc.Service
}

func NewService(q *db.Queries, orgSvc *orgsvc.Service, slotSvc *slotsvc.Service) *Service {
	return &Service{q: q, orgSvc: orgSvc, slotSvc: slotSvc}
}

func (s *Service) Create(ctx context.Context, ownerID pgtype.UUID, in Input) (db.InterviewType, error) {
	if !ownerID.Valid {
		return db.InterviewType{}, ErrForbidden
	}

	in, err := validateFields(in)
	if err != nil {
		return db.InterviewType{}, err
	}

	org, err := s.orgSvc.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, orgsvc.ErrOrgNotFound) {
			return db.InterviewType{}, ErrOrgRequired
		}
		return db.InterviewType{}, err
	}

	if err := s.enforceInterviewTypeLimit(ctx, org); err != nil {
		return db.InterviewType{}, err
	}

	if err := s.ensureGoogleConnected(ctx, ownerID, in.MeetingProvider); err != nil {
		return db.InterviewType{}, err
	}

	if err := s.ensureSlugAvailable(ctx, org.ID, in.Slug, pgtype.UUID{}); err != nil {
		return db.InterviewType{}, err
	}

	meetingURL, hasURL := optionalMeetingURL(in.MeetingProvider, in.MeetingURL)

	created, err := s.q.CreateInterviewType(ctx, db.CreateInterviewTypeParams{
		OrganizationID:  org.ID,
		Title:           in.Title,
		Slug:            in.Slug,
		DurationMinutes: in.DurationMinutes,
		BufferMinutes:   in.BufferMinutes,
		MeetingProvider: in.MeetingProvider,
		MeetingUrl:      textFromOptional(meetingURL, hasURL),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.InterviewType{}, ErrSlugTaken
		}
		return db.InterviewType{}, err
	}

	if s.slotSvc != nil {
		if err := s.slotSvc.RegenerateForType(ctx, nil, org.ID, created.ID, created.DurationMinutes, created.BufferMinutes); err != nil {
			return db.InterviewType{}, err
		}
	}

	return created, nil
}

func (s *Service) List(ctx context.Context, ownerID pgtype.UUID) ([]db.InterviewType, error) {
	if !ownerID.Valid {
		return nil, ErrForbidden
	}

	org, err := s.orgSvc.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, orgsvc.ErrOrgNotFound) {
			return nil, ErrOrgRequired
		}
		return nil, err
	}

	return s.q.ListInterviewTypesByOrg(ctx, org.ID)
}

func (s *Service) Update(ctx context.Context, id, ownerID pgtype.UUID, in Input) (db.InterviewType, error) {
	if !ownerID.Valid {
		return db.InterviewType{}, ErrForbidden
	}

	in, err := validateFields(in)
	if err != nil {
		return db.InterviewType{}, err
	}

	org, err := s.orgSvc.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, orgsvc.ErrOrgNotFound) {
			return db.InterviewType{}, ErrOrgRequired
		}
		return db.InterviewType{}, err
	}

	existing, err := s.getForOrg(ctx, id, org.ID)
	if err != nil {
		return db.InterviewType{}, err
	}

	if err := s.ensureGoogleConnected(ctx, ownerID, in.MeetingProvider); err != nil {
		return db.InterviewType{}, err
	}

	if in.Slug != existing.Slug {
		if err := s.ensureSlugAvailable(ctx, org.ID, in.Slug, id); err != nil {
			return db.InterviewType{}, err
		}
	}

	meetingURL, hasURL := optionalMeetingURL(in.MeetingProvider, in.MeetingURL)

	updated, err := s.q.UpdateInterviewType(ctx, db.UpdateInterviewTypeParams{
		ID:              id,
		Title:           in.Title,
		Slug:            in.Slug,
		DurationMinutes: in.DurationMinutes,
		BufferMinutes:   in.BufferMinutes,
		MeetingProvider: in.MeetingProvider,
		MeetingUrl:      textFromOptional(meetingURL, hasURL),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return db.InterviewType{}, ErrSlugTaken
		}
		return db.InterviewType{}, err
	}

	if s.slotSvc != nil &&
		(in.DurationMinutes != existing.DurationMinutes || in.BufferMinutes != existing.BufferMinutes) {
		if err := s.slotSvc.RegenerateForType(ctx, nil, org.ID, updated.ID, updated.DurationMinutes, updated.BufferMinutes); err != nil {
			return db.InterviewType{}, err
		}
	}

	return updated, nil
}

func (s *Service) getForOrg(ctx context.Context, id, orgID pgtype.UUID) (db.InterviewType, error) {
	item, err := s.q.GetInterviewTypeByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.InterviewType{}, ErrNotFound
		}
		return db.InterviewType{}, err
	}

	if !item.OrganizationID.Valid || !orgID.Valid || item.OrganizationID.Bytes != orgID.Bytes {
		return db.InterviewType{}, ErrForbidden
	}

	return item, nil
}

func (s *Service) enforceInterviewTypeLimit(ctx context.Context, org db.Organization) error {
	if org.Plan != db.PlanTypeFree {
		return nil
	}

	count, err := s.q.CountInterviewTypesByOrg(ctx, org.ID)
	if err != nil {
		return err
	}
	if count >= freePlanMaxTypes {
		return ErrPlanLimitInterviewTypes
	}
	return nil
}

func (s *Service) ensureGoogleConnected(ctx context.Context, ownerID pgtype.UUID, provider db.MeetingProvider) error {
	if provider != db.MeetingProviderGoogleMeet {
		return nil
	}

	connected, err := s.q.IsGoogleConnected(ctx, ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGoogleNotConnected
		}
		return err
	}

	if !connected {
		return ErrGoogleNotConnected
	}
	return nil
}

func (s *Service) ensureSlugAvailable(ctx context.Context, orgID pgtype.UUID, slug string, excludeID pgtype.UUID) error {
	exists, err := s.q.InterviewTypeExistsBySlug(ctx, db.InterviewTypeExistsBySlugParams{
		OrganizationID: orgID,
		Slug:           slug,
	})
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if excludeID.Valid {
		existing, err := s.q.GetInterviewTypeByID(ctx, excludeID)
		if err == nil && existing.Slug == slug {
			return nil
		}
	}

	return ErrSlugTaken
}

func textFromOptional(value string, valid bool) pgtype.Text {
	if !valid {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
