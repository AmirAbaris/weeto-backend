package organization

import (
	"context"
	"errors"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/plan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type UsageCounter struct {
	Used  int32
	Limit *int32
}

type MeetLinksUsage struct {
	Used        int32
	Limit       *int32
	PeriodStart time.Time
}

type OrganizationUsage struct {
	MeetLinks      MeetLinksUsage
	InterviewTypes UsageCounter
}

type OrganizationWithUsage struct {
	Organization db.Organization
	Usage        OrganizationUsage
}

func (s *Service) UsageForOrg(ctx context.Context, org db.Organization) (OrganizationUsage, error) {
	typeCount, err := s.q.CountInterviewTypesByOrg(ctx, org.ID)
	if err != nil {
		return OrganizationUsage{}, err
	}

	periodStart := time.Time{}
	if org.MeetLinksPeriodStart.Valid {
		periodStart = org.MeetLinksPeriodStart.Time
	}

	return OrganizationUsage{
		MeetLinks: MeetLinksUsage{
			Used:        org.MeetLinksUsed,
			Limit:       plan.MaxMeetLinksPerMonth(org.Plan),
			PeriodStart: periodStart,
		},
		InterviewTypes: UsageCounter{
			Used:  typeCount,
			Limit: plan.MaxInterviewTypes(org.Plan),
		},
	}, nil
}

func (s *Service) GetByOwnerWithUsage(ctx context.Context, ownerID pgtype.UUID) (OrganizationWithUsage, error) {
	org, err := s.GetByOwner(ctx, ownerID)
	if err != nil {
		return OrganizationWithUsage{}, err
	}
	usage, err := s.UsageForOrg(ctx, org)
	if err != nil {
		return OrganizationWithUsage{}, err
	}
	return OrganizationWithUsage{Organization: org, Usage: usage}, nil
}

func (s *Service) GetByIDWithUsage(ctx context.Context, id, ownerID pgtype.UUID) (OrganizationWithUsage, error) {
	org, err := s.GetByID(ctx, id, ownerID)
	if err != nil {
		return OrganizationWithUsage{}, err
	}
	usage, err := s.UsageForOrg(ctx, org)
	if err != nil {
		return OrganizationWithUsage{}, err
	}
	return OrganizationWithUsage{Organization: org, Usage: usage}, nil
}

func (s *Service) GetByIDWithUsageAdmin(ctx context.Context, id pgtype.UUID) (OrganizationWithUsage, error) {
	if !id.Valid {
		return OrganizationWithUsage{}, ErrInvalidOwner
	}
	org, err := s.q.GetOrganizationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrganizationWithUsage{}, ErrOrgNotFound
		}
		return OrganizationWithUsage{}, err
	}
	usage, err := s.UsageForOrg(ctx, org)
	if err != nil {
		return OrganizationWithUsage{}, err
	}
	return OrganizationWithUsage{Organization: org, Usage: usage}, nil
}
