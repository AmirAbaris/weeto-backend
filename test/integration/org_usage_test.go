package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	interviewtypesvc "github.com/AmirAbaris/weeto-backend/internal/service/interviewtype"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
)

func TestGetOrganizationWithUsage(t *testing.T) {
	env := NewTestEnv(t, time.Now())
	cfg := &config.Config{JWTSecret: "integration-test-secret"}
	orgSvc := orgsvc.NewService(env.Queries, cfg)

	withUsage, err := orgSvc.GetByOwnerWithUsage(env.Ctx, env.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if withUsage.Organization.Plan != db.PlanTypeFree {
		t.Fatalf("plan = %q, want free", withUsage.Organization.Plan)
	}
	if withUsage.Usage.InterviewTypes.Used != 0 {
		t.Fatalf("interview types used = %d, want 0", withUsage.Usage.InterviewTypes.Used)
	}
	if withUsage.Usage.InterviewTypes.Limit == nil || *withUsage.Usage.InterviewTypes.Limit != 3 {
		t.Fatalf("interview types limit = %v, want 3", withUsage.Usage.InterviewTypes.Limit)
	}
	if withUsage.Usage.MeetLinks.Used != 0 {
		t.Fatalf("meet links used = %d, want 0", withUsage.Usage.MeetLinks.Used)
	}
	if withUsage.Usage.MeetLinks.Limit == nil || *withUsage.Usage.MeetLinks.Limit != 15 {
		t.Fatalf("meet links limit = %v, want 15", withUsage.Usage.MeetLinks.Limit)
	}
}

func TestDowngradeToFreeGrandfathersInterviewTypes(t *testing.T) {
	env := NewTestEnv(t, time.Now())
	cfg := &config.Config{JWTSecret: "integration-test-secret"}
	orgSvc := orgsvc.NewService(env.Queries, cfg)
	itSvc := interviewtypesvc.NewService(env.Queries, orgSvc, env.SlotSvc)

	if _, err := orgSvc.UpdatePlan(env.Ctx, env.OrgID, db.PlanTypePro); err != nil {
		t.Fatal(err)
	}

	meetingURL := "تهران"
	for i := range 4 {
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

	downgraded, err := orgSvc.UpdatePlan(env.Ctx, env.OrgID, db.PlanTypeFree)
	if err != nil {
		t.Fatalf("downgrade: %v", err)
	}
	if downgraded.Plan != db.PlanTypeFree {
		t.Fatalf("plan = %q, want free", downgraded.Plan)
	}

	count, err := env.Queries.CountInterviewTypesByOrg(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("interview type count = %d, want 4 (grandfathered)", count)
	}

	_, err = itSvc.Create(env.Ctx, env.UserID, interviewtypesvc.Input{
		Title:           "Type 5",
		Slug:            "type-5",
		DurationMinutes: 30,
		BufferMinutes:   0,
		MeetingProvider: db.MeetingProviderOnSite,
		MeetingURL:      &meetingURL,
	})
	if !errors.Is(err, interviewtypesvc.ErrPlanLimitInterviewTypes) {
		t.Fatalf("err = %v, want ErrPlanLimitInterviewTypes", err)
	}
}

func TestProPlanUnlimitedMeetLinks(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)
	cfg := &config.Config{JWTSecret: "integration-test-secret"}
	orgSvc := orgsvc.NewService(env.Queries, cfg)

	if _, err := orgSvc.UpdatePlan(env.Ctx, env.OrgID, db.PlanTypePro); err != nil {
		t.Fatal(err)
	}
	env.SetMeetLinksUsed(15)

	it := env.CreateGoogleMeetInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	_, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Ali Rezaei",
		Phone:  "+989121234567",
		Email:  "ali@example.com",
	})
	if err != nil {
		t.Fatalf("book on pro after 15 used: %v", err)
	}

	org, err := env.Queries.GetOrganizationByID(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatal(err)
	}
	if org.MeetLinksUsed != 16 {
		t.Fatalf("meet_links_used = %d, want 16", org.MeetLinksUsed)
	}
}

func TestMeetLinksPeriodResetsMonthly(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)
	env.SetMeetLinksUsed(15)
	env.SetMeetLinksPeriodStart(now.AddDate(0, -1, 0))

	it := env.CreateGoogleMeetInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	_, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Ali Rezaei",
		Phone:  "+989121234567",
		Email:  "ali@example.com",
	})
	if err != nil {
		t.Fatalf("book after monthly reset: %v", err)
	}

	org, err := env.Queries.GetOrganizationByID(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatal(err)
	}
	if org.MeetLinksUsed != 1 {
		t.Fatalf("meet_links_used = %d, want 1 after reset", org.MeetLinksUsed)
	}
}

func TestGoogleMeetCancelDoesNotRefundQuota(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateGoogleMeetInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Ali Rezaei",
		Phone:  "+989121234567",
		Email:  "ali@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	if err := env.BookingSvc.Cancel(env.Ctx, env.UserID, result.Booking.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	org, err := env.Queries.GetOrganizationByID(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatal(err)
	}
	if org.MeetLinksUsed != 1 {
		t.Fatalf("meet_links_used = %d, want 1 after cancel (no refund)", org.MeetLinksUsed)
	}
}

func TestHTTPGetOrganizationMeReturnsUsage(t *testing.T) {
	env := NewTestEnv(t, time.Now())
	ts := newTestServer(t, env)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/organizations/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+IssueTestToken(t, env.UserID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		t.Fatalf("usage missing: %#v", payload)
	}
	interviewTypes, ok := usage["interview_types"].(map[string]any)
	if !ok {
		t.Fatalf("interview_types missing: %#v", usage)
	}
	if interviewTypes["used"] == nil || interviewTypes["limit"] == nil {
		t.Fatalf("interview_types counters missing: %#v", interviewTypes)
	}
}

func TestHTTPPlanLimitInterviewTypes(t *testing.T) {
	env := NewTestEnv(t, time.Now())
	cfg := &config.Config{JWTSecret: "integration-test-secret"}
	orgSvc := orgsvc.NewService(env.Queries, cfg)
	itSvc := interviewtypesvc.NewService(env.Queries, orgSvc, env.SlotSvc)
	meetingURL := "تهران"

	for i := range 3 {
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

	ts := newTestServer(t, env)
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"title":             "Type 4",
		"slug":              "type-4",
		"duration_minutes":  30,
		"buffer_minutes":    0,
		"meeting_provider":  "on_site",
		"meeting_url":       meetingURL,
	})
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/interview-types", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+IssueTestToken(t, env.UserID))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}

	var detail map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail["code"] != "plan_limit_interview_types" {
		t.Fatalf("code = %v, want plan_limit_interview_types", detail["code"])
	}
	if detail["action_url"] != "/dashboard/settings#plan" {
		t.Fatalf("action_url = %v", detail["action_url"])
	}
}

func TestHTTPPlanLimitMeetLinks(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)
	env.SetMeetLinksUsed(15)

	it := env.CreateGoogleMeetInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	ts := newTestServer(t, env)
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"slot_id": slot.ID.String(),
		"name":    "Ali Rezaei",
		"phone":   "+989121234567",
		"email":   "ali@example.com",
	})
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/public/"+env.OrgSlug+"/"+it.Slug+"/book", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, raw)
	}

	var detail map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail["code"] != "plan_limit_meet_links" {
		t.Fatalf("code = %v, want plan_limit_meet_links", detail["code"])
	}
	if detail["action"] != "contact" {
		t.Fatalf("action = %v, want contact", detail["action"])
	}
}
