package integration

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	interviewtypesvc "github.com/AmirAbaris/weeto-backend/internal/service/interviewtype"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
)

func TestUpdateOrgPreservesPlan(t *testing.T) {
	env := NewTestEnv(t, time.Now())
	cfg := &config.Config{JWTSecret: "integration-test-secret"}
	orgSvc := orgsvc.NewService(env.Queries, cfg)

	org, err := orgSvc.GetByID(env.Ctx, env.OrgID, env.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if org.Plan != db.PlanTypeFree {
		t.Fatalf("plan = %q, want free", org.Plan)
	}

	updated, err := orgSvc.UpdateOrg(env.Ctx, env.OrgID, env.UserID, "Renamed Org", org.Slug, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Plan != db.PlanTypeFree {
		t.Fatalf("plan = %q, want free after rename", updated.Plan)
	}
	if updated.Name != "Renamed Org" {
		t.Fatalf("name = %q, want Renamed Org", updated.Name)
	}
}

func TestAdminUpdatePlanUnlocksInterviewTypeLimit(t *testing.T) {
	env := NewTestEnv(t, time.Now())
	cfg := &config.Config{JWTSecret: "integration-test-secret"}
	orgSvc := orgsvc.NewService(env.Queries, cfg)
	itSvc := interviewtypesvc.NewService(env.Queries, orgSvc, env.SlotSvc)

	for i := range 3 {
		meetingURL := "تهران"
		_, err := itSvc.Create(env.Ctx, env.UserID, interviewtypesvc.Input{
			Title:           fmt.Sprintf("Type %d", i+1),
			Slug:            fmt.Sprintf("type-%d", i+1),
			DurationMinutes: 30,
			BufferMinutes:   0,
			MeetingProvider: db.MeetingProviderOnSite,
			MeetingURL:      &meetingURL,
		})
		if err != nil {
			t.Fatalf("create type %d: %v", i+1, err)
		}
	}

	meetingURL := "تهران"
	_, err := itSvc.Create(env.Ctx, env.UserID, interviewtypesvc.Input{
		Title:           "Type 4",
		Slug:            "type-4",
		DurationMinutes: 30,
		BufferMinutes:   0,
		MeetingProvider: db.MeetingProviderOnSite,
		MeetingURL:      &meetingURL,
	})
	if !errors.Is(err, interviewtypesvc.ErrPlanLimitInterviewTypes) {
		t.Fatalf("err = %v, want ErrPlanLimitInterviewTypes", err)
	}

	updated, err := orgSvc.UpdatePlan(env.Ctx, env.OrgID, db.PlanTypePro)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Plan != db.PlanTypePro {
		t.Fatalf("plan = %q, want pro", updated.Plan)
	}

	_, err = itSvc.Create(env.Ctx, env.UserID, interviewtypesvc.Input{
		Title:           "Type 4",
		Slug:            "type-4",
		DurationMinutes: 30,
		BufferMinutes:   0,
		MeetingProvider: db.MeetingProviderOnSite,
		MeetingURL:      &meetingURL,
	})
	if err != nil {
		t.Fatalf("create 4th type after upgrade: %v", err)
	}
}
