package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
)

func TestListBookingsTodayAndUpcoming(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.MonFri9to17(50))

	todaySlot := env.FirstAvailableSlot(it)
	tuesday := time.Date(2026, 7, 7, 0, 0, 0, 0, loc)
	tuesdaySlots := env.SlotsOnLocalDay(it.ID, tuesday, loc)
	if len(tuesdaySlots) == 0 {
		t.Fatal("expected tuesday slots")
	}
	upcomingSlot := tuesdaySlots[0]

	_, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: todaySlot.ID,
		Name:   "Ali Rezaei",
		Phone:  "+989121234567",
		Email:  "ali@example.com",
	})
	if err != nil {
		t.Fatalf("book today: %v", err)
	}

	_, err = env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: upcomingSlot.ID,
		Name:   "Sara Ahmadi",
		Phone:  "+989129876543",
		Email:  "sara@example.com",
	})
	if err != nil {
		t.Fatalf("book upcoming: %v", err)
	}

	ts := newTestServer(t, env)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/bookings", nil)
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

	var result struct {
		Today    []map[string]any `json:"today"`
		Upcoming []map[string]any `json:"upcoming"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	if len(result.Today) != 1 {
		t.Fatalf("today count = %d, want 1", len(result.Today))
	}
	if len(result.Upcoming) != 1 {
		t.Fatalf("upcoming count = %d, want 1", len(result.Upcoming))
	}

	if result.Today[0]["name"] != "Ali Rezaei" {
		t.Fatalf("today name = %v", result.Today[0]["name"])
	}
	if result.Today[0]["phone"] != "+989121234567" {
		t.Fatalf("today phone = %v", result.Today[0]["phone"])
	}
	if result.Today[0]["email"] != "ali@example.com" {
		t.Fatalf("today email = %v", result.Today[0]["email"])
	}

	if result.Upcoming[0]["name"] != "Sara Ahmadi" {
		t.Fatalf("upcoming name = %v", result.Upcoming[0]["name"])
	}
}

func TestListBookingsRequiresAuth(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	ts := newTestServer(t, env)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/bookings")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestCancelBookingFreesSlot(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Cancel Me",
		Phone:  "+989121234567",
		Email:  "cancel@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	ts := newTestServer(t, env)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/bookings/"+result.Booking.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+IssueTestToken(t, env.UserID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	stored, err := env.Queries.GetBookingByID(env.Ctx, result.Booking.ID)
	if err != nil {
		t.Fatalf("get booking: %v", err)
	}
	if stored.Status != db.BookingStatusCancelled {
		t.Fatalf("status = %q, want cancelled", stored.Status)
	}

	freed, err := env.Queries.GetSlotByID(env.Ctx, slot.ID)
	if err != nil {
		t.Fatalf("get slot: %v", err)
	}
	if freed.Booked {
		t.Fatal("expected slot to be unbooked")
	}

	outbox, err := env.Queries.ListNotificationOutboxByOrg(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}

	var cancelled int
	for _, row := range outbox {
		if row.EventType == db.NotificationEventTypeBookingCancelled {
			cancelled++
		}
	}
	if cancelled != 2 {
		t.Fatalf("booking_cancelled outbox rows = %d, want 2", cancelled)
	}

	second, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Rebooked",
		Phone:  "+989129876543",
		Email:  "rebook@example.com",
	})
	if err != nil {
		t.Fatalf("rebook: %v", err)
	}
	if second.Booking.ID == result.Booking.ID {
		t.Fatal("expected a new booking row after rebook")
	}
}

func TestCancelBookingNotFound(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	ts := newTestServer(t, env)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/bookings/00000000-0000-4000-8000-000000000001", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+IssueTestToken(t, env.UserID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestCancelBookingWrongOrg(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Protected",
		Phone:  "+989121234567",
		Email:  "protected@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	otherUser := seedUser(t, env.Queries)
	otherToken := IssueTestToken(t, otherUser)

	ts := newTestServer(t, env)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/bookings/"+result.Booking.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+otherToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestCancelBookingInvalidID(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	ts := newTestServer(t, env)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/bookings/not-a-uuid", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+IssueTestToken(t, env.UserID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
