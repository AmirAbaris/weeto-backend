package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	authhandler "github.com/AmirAbaris/weeto-backend/internal/handler/auth"
	docshandler "github.com/AmirAbaris/weeto-backend/internal/handler/docs"
	"github.com/AmirAbaris/weeto-backend/internal/handler/health"
	availabilityhandler "github.com/AmirAbaris/weeto-backend/internal/handler/availability"
	bookinghandler "github.com/AmirAbaris/weeto-backend/internal/handler/booking"
	interviewtypehandler "github.com/AmirAbaris/weeto-backend/internal/handler/interviewtype"
	orghandler "github.com/AmirAbaris/weeto-backend/internal/handler/organization"
	publichandler "github.com/AmirAbaris/weeto-backend/internal/handler/public"
	"github.com/AmirAbaris/weeto-backend/internal/server"
	authsvc "github.com/AmirAbaris/weeto-backend/internal/service/auth"
	availabilitysvc "github.com/AmirAbaris/weeto-backend/internal/service/availability"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	interviewtypesvc "github.com/AmirAbaris/weeto-backend/internal/service/interviewtype"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
)

func TestBookSuccess(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
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

	stored, err := env.Queries.GetBookingByID(env.Ctx, result.Booking.ID)
	if err != nil {
		t.Fatalf("get booking: %v", err)
	}
	if stored.CandidateName != "Ali Rezaei" {
		t.Fatalf("candidate name = %q", stored.CandidateName)
	}
	if stored.Status != db.BookingStatusScheduled {
		t.Fatalf("status = %q, want scheduled", stored.Status)
	}

	count, err := env.Queries.CountBookingsBySlot(env.Ctx, slot.ID)
	if err != nil {
		t.Fatalf("count bookings: %v", err)
	}
	if count != 1 {
		t.Fatalf("booking count = %d, want 1", count)
	}

	updatedSlot, err := env.Queries.ListSlotsByTypeInWindow(env.Ctx, db.ListSlotsByTypeInWindowParams{
		InterviewTypeID: it.ID,
		StartAt:         slot.StartAt,
		StartAt_2:       slot.EndAt,
	})
	if err != nil {
		t.Fatalf("list slots: %v", err)
	}
	if len(updatedSlot) != 1 || !updatedSlot[0].Booked {
		t.Fatal("expected slot to be booked")
	}

	outbox, err := env.Queries.ListNotificationOutboxByOrg(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	if len(outbox) != 2 {
		t.Fatalf("outbox rows = %d, want 2", len(outbox))
	}
	for _, row := range outbox {
		if row.EventType != db.NotificationEventTypeBookingCreated {
			t.Fatalf("event type = %q", row.EventType)
		}
		if row.Status != db.NotificationStatusPending {
			t.Fatalf("status = %q", row.Status)
		}
	}

	recipients := map[string]int{}
	for _, row := range outbox {
		var payload map[string]any
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		recipient, _ := payload["recipient"].(string)
		recipients[recipient]++
	}
	if recipients["recruiter"] != 1 || recipients["candidate"] != 1 {
		t.Fatalf("recipients = %#v", recipients)
	}
}

func TestOnSiteBooking(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
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

	if result.Booking.MeetLink.Valid {
		t.Fatalf("meet_link should be empty for on_site, got %q", result.Booking.MeetLink.String)
	}

	outbox, err := env.Queries.ListNotificationOutboxByOrg(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	if len(outbox) != 2 {
		t.Fatalf("outbox rows = %d, want 2", len(outbox))
	}

	var payload map[string]any
	if err := json.Unmarshal(outbox[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["meet_link"] != nil {
		t.Fatalf("payload meet_link = %v, want nil", payload["meet_link"])
	}
	if payload["recipient"] != "recruiter" && payload["recipient"] != "candidate" {
		t.Fatalf("recipient = %v", payload["recipient"])
	}
	loc, ok := payload["meeting_location"].(string)
	if !ok || loc != "تهران، خیابان ولیعصر، پلاک ۱۲" {
		t.Fatalf("meeting_location = %v, want on-site address", payload["meeting_location"])
	}
}

func TestBookConcurrentConflict(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	input := bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Sara Ahmadi",
		Phone:  "+989121234567",
		Email:  "sara@example.com",
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, input)
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var successes, conflicts int
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, bookingsvc.ErrSlotUnavailable):
			conflicts++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}

	count, err := env.Queries.CountBookingsBySlot(env.Ctx, slot.ID)
	if err != nil {
		t.Fatalf("count bookings: %v", err)
	}
	if count != 1 {
		t.Fatalf("booking count = %d, want 1", count)
	}
}

func TestBookHTTPConcurrent409(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	ts := newTestServer(t, env)
	defer ts.Close()

	body, err := json.Marshal(map[string]string{
		"slot_id": slot.ID.String(),
		"name":    "HTTP Candidate",
		"phone":   "+989121234567",
		"email":   "http@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	url := ts.URL + "/public/" + env.OrgSlug + "/" + it.Slug + "/book"
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var wg sync.WaitGroup

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			resp, err := http.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				t.Errorf("post: %v", err)
				statuses <- 0
				return
			}
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)
			statuses <- resp.StatusCode
		}()
	}

	close(start)
	wg.Wait()
	close(statuses)

	var created, conflict int
	for code := range statuses {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("unexpected status: %d", code)
		}
	}

	if created != 1 || conflict != 1 {
		t.Fatalf("created=%d conflict=%d, want 1 and 1", created, conflict)
	}
}

func newTestServer(t *testing.T, env *TestEnv) *httptest.Server {
	t.Helper()

	cfg := &config.Config{JWTSecret: "integration-test-secret"}
	queries := env.Queries
	orgSvc := orgsvc.NewService(queries, cfg)
	slotSvc := env.SlotSvc
	itSvc := interviewtypesvc.NewService(queries, orgSvc, slotSvc)
	availSvc := availabilitysvc.NewService(env.Pool, queries, orgSvc, slotSvc)
	bookingSvc := bookingsvc.NewService(env.Pool, queries, orgSvc, slotSvc, env.Calendar)
	authSvc := authsvc.NewService(queries, cfg)

	mux := http.NewServeMux()
	server.Register(mux, cfg.JWTSecret, server.Handlers{
		Health:        health.NewHandler(),
		Docs:          docshandler.NewHandler(),
		Auth:          authhandler.NewHandler(authSvc),
		Organization:  orghandler.NewHandler(orgSvc),
		InterviewType: interviewtypehandler.NewHandler(itSvc),
		Availability:  availabilityhandler.NewHandler(availSvc),
		Booking:       bookinghandler.NewHandler(bookingSvc),
		Public:        publichandler.NewHandler(bookingSvc),
	})

	return httptest.NewServer(mux)
}
