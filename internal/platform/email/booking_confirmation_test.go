package email_test

import (
	"strings"
	"testing"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/platform/email"
)

func TestParseBookingConfirmationPayload_BuildsMagicLinks(t *testing.T) {
	payload := []byte(`{
		"candidate_name": "Ali",
		"candidate_email": "ali@example.com",
		"organization_name": "Acme",
		"interview_type_title": "Backend Interview",
		"slot_start_at": "2026-07-07T10:00:00Z",
		"slot_end_at": "2026-07-07T11:00:00Z",
		"reschedule_token": "reschedule-token",
		"cancel_token": "cancel-token",
		"meet_link": "https://meet.google.com/abc"
	}`)

	data, err := email.ParseBookingConfirmationPayload(payload, "http://localhost:3000/")
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}

	if data.RescheduleURL != "http://localhost:3000/reschedule/reschedule-token" {
		t.Fatalf("reschedule url = %q", data.RescheduleURL)
	}
	if data.CancelURL != "http://localhost:3000/cancel/cancel-token" {
		t.Fatalf("cancel url = %q", data.CancelURL)
	}
	if data.MeetLink != "https://meet.google.com/abc" {
		t.Fatalf("meet link = %q", data.MeetLink)
	}
}

func TestBookingConfirmationMessage_IncludesLinks(t *testing.T) {
	msg := email.BookingConfirmationMessage(email.BookingConfirmationData{
		CandidateName:      "Ali",
		CandidateEmail:     "ali@example.com",
		OrganizationName:   "Acme",
		InterviewTypeTitle: "Backend Interview",
		SlotStartAt:        time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		SlotEndAt:          time.Date(2026, 7, 7, 11, 0, 0, 0, time.UTC),
		MeetLink:           "https://meet.google.com/abc",
		RescheduleURL:      "http://localhost:3000/reschedule/reschedule-token",
		CancelURL:          "http://localhost:3000/cancel/cancel-token",
	})

	if msg.To != "ali@example.com" {
		t.Fatalf("to = %q", msg.To)
	}
	if !strings.Contains(msg.HTML, "http://localhost:3000/reschedule/reschedule-token") {
		t.Fatalf("html missing reschedule link: %s", msg.HTML)
	}
	if !strings.Contains(msg.HTML, "http://localhost:3000/cancel/cancel-token") {
		t.Fatalf("html missing cancel link: %s", msg.HTML)
	}
	if !strings.Contains(msg.HTML, `dir="rtl"`) {
		t.Fatalf("html missing rtl direction")
	}
}
