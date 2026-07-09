package orgresponse

import (
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
)

type usageCounterResponse struct {
	Used  int32  `json:"used"`
	Limit *int32 `json:"limit"`
}

type meetLinksUsageResponse struct {
	Used        int32     `json:"used"`
	Limit       *int32    `json:"limit"`
	PeriodStart time.Time `json:"period_start,omitempty"`
}

type organizationUsageResponse struct {
	MeetLinks      meetLinksUsageResponse `json:"meet_links"`
	InterviewTypes usageCounterResponse   `json:"interview_types"`
}

type OrganizationResponse struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Slug      string                     `json:"slug"`
	LogoURL   *string                    `json:"logo_url,omitempty"`
	OwnerID   string                     `json:"owner_id"`
	Plan      string                     `json:"plan"`
	Usage     organizationUsageResponse  `json:"usage"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

func FromOrganization(org db.Organization, usage orgsvc.OrganizationUsage) OrganizationResponse {
	resp := OrganizationResponse{
		Name: org.Name,
		Slug: org.Slug,
		Plan: string(org.Plan),
		Usage: organizationUsageResponse{
			MeetLinks: meetLinksUsageResponse{
				Used:  usage.MeetLinks.Used,
				Limit: usage.MeetLinks.Limit,
			},
			InterviewTypes: usageCounterResponse{
				Used:  usage.InterviewTypes.Used,
				Limit: usage.InterviewTypes.Limit,
			},
		},
	}
	if !usage.MeetLinks.PeriodStart.IsZero() {
		resp.Usage.MeetLinks.PeriodStart = usage.MeetLinks.PeriodStart
	}
	if org.ID.Valid {
		resp.ID = org.ID.String()
	}
	if org.OwnerID.Valid {
		resp.OwnerID = org.OwnerID.String()
	}
	if org.LogoUrl.Valid {
		logo := org.LogoUrl.String
		resp.LogoURL = &logo
	}
	if org.CreatedAt.Valid {
		resp.CreatedAt = org.CreatedAt.Time
	}
	if org.UpdatedAt.Valid {
		resp.UpdatedAt = org.UpdatedAt.Time
	}
	return resp
}

func FromWithUsage(item orgsvc.OrganizationWithUsage) OrganizationResponse {
	return FromOrganization(item.Organization, item.Usage)
}
