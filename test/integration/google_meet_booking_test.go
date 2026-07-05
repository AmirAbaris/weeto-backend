package integration

import (
	"errors"
	"testing"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
)

func TestGoogleMeetBookSuccess(t *testing.T) {
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

	stored, err := env.Queries.GetBookingByID(env.Ctx, result.Booking.ID)
	if err != nil {
		t.Fatalf("get booking: %v", err)
	}
	if !stored.MeetLink.Valid || stored.MeetLink.String != "https://meet.google.com/mock-link" {
		t.Fatalf("meet_link = %+v", stored.MeetLink)
	}
	if !stored.CalendarEventID.Valid || stored.CalendarEventID.String != "mock-event-id" {
		t.Fatalf("calendar_event_id = %+v", stored.CalendarEventID)
	}
	if len(env.Calendar.Created) != 1 {
		t.Fatalf("calendar create calls = %d, want 1", len(env.Calendar.Created))
	}

	org, err := env.Queries.GetOrganizationByID(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatalf("get org: %v", err)
	}
	if org.MeetLinksUsed != 1 {
		t.Fatalf("meet_links_used = %d, want 1", org.MeetLinksUsed)
	}
}

func TestGoogleMeetBookNotConnected(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	slug := "google-meet-no-connect"
	it, err := env.Queries.CreateInterviewType(env.Ctx, db.CreateInterviewTypeParams{
		OrganizationID:  env.OrgID,
		Title:           "Google Meet Interview",
		Slug:            slug,
		DurationMinutes: 60,
		BufferMinutes:   0,
		MeetingProvider: db.MeetingProviderGoogleMeet,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := env.SlotSvc.RegenerateForType(env.Ctx, nil, env.OrgID, it.ID, 60, 0); err != nil {
		t.Fatal(err)
	}
	env.UpsertAvailability(fixtures.Monday9to17(50))

	slot := env.FirstAvailableSlot(it)
	_, err = env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Ali Rezaei",
		Phone:  "+989121234567",
		Email:  "ali@example.com",
	})
	if !errors.Is(err, bookingsvc.ErrGoogleNotConnected) {
		t.Fatalf("err = %v, want ErrGoogleNotConnected", err)
	}

	count, err := env.Queries.CountBookingsBySlot(env.Ctx, slot.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("booking count = %d, want 0", count)
	}
}

func TestGoogleMeetPlanLimit(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)
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
	if !errors.Is(err, bookingsvc.ErrMeetLinkLimitReached) {
		t.Fatalf("err = %v, want ErrMeetLinkLimitReached", err)
	}

	count, err := env.Queries.CountBookingsBySlot(env.Ctx, slot.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("booking count = %d, want 0", count)
	}
}

func TestGoogleMeetCalendarFailureRollsBack(t *testing.T) {
	now := mondayMorningTehran(t)
	calendar := &MockCalendarClient{CreateFn: calendarCreateError()}
	env := NewTestEnvWithCalendar(t, now, calendar)

	it := env.CreateGoogleMeetInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	slot := env.FirstAvailableSlot(it)
	_, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Ali Rezaei",
		Phone:  "+989121234567",
		Email:  "ali@example.com",
	})
	if !errors.Is(err, bookingsvc.ErrGoogleCalendarFailed) {
		t.Fatalf("err = %v, want ErrGoogleCalendarFailed", err)
	}

	count, err := env.Queries.CountBookingsBySlot(env.Ctx, slot.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("booking count = %d, want 0 after rollback", count)
	}

	org, err := env.Queries.GetOrganizationByID(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatal(err)
	}
	if org.MeetLinksUsed != 0 {
		t.Fatalf("meet_links_used = %d, want 0 after decrement", org.MeetLinksUsed)
	}
}

func TestGoogleMeetCancelDeletesCalendarEvent(t *testing.T) {
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

	if len(env.Calendar.Deleted) != 1 || env.Calendar.Deleted[0] != "mock-event-id" {
		t.Fatalf("deleted events = %v", env.Calendar.Deleted)
	}
}

func TestGoogleMeetPlanLimitSlotFreed(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)
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
	if !errors.Is(err, bookingsvc.ErrMeetLinkLimitReached) {
		t.Fatalf("err = %v", err)
	}

	updated, err := env.Queries.ListSlotsByTypeInWindow(env.Ctx, db.ListSlotsByTypeInWindowParams{
		InterviewTypeID: it.ID,
		StartAt:         slot.StartAt,
		StartAt_2:       slot.EndAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated) != 1 || updated[0].Booked {
		t.Fatal("expected slot to remain available after plan limit rollback")
	}
}
