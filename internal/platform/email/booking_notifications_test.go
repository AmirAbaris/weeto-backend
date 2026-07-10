package email_test

import (
	"strings"
	"testing"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/platform/email"
)

func TestBookingRescheduledMessage_IncludesNewAndPreviousTime(t *testing.T) {
	msg := email.BookingRescheduledMessage(email.BookingNotificationData{
		CandidateName:      "Ali",
		CandidateEmail:     "ali@example.com",
		OrganizationName:   "Acme",
		InterviewTypeTitle: "Backend Interview",
		SlotStartAt:        time.Date(2026, 7, 7, 11, 0, 0, 0, time.UTC),
		SlotEndAt:          time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC),
		PreviousSlotStart:  time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		PreviousSlotEnd:    time.Date(2026, 7, 7, 11, 0, 0, 0, time.UTC),
		HasPreviousSlot:    true,
		RescheduleURL:      "http://localhost:3000/reschedule/token",
		CancelURL:          "http://localhost:3000/cancel/token",
	})

	if !strings.Contains(msg.HTML, "زمان قبلی") {
		t.Fatalf("html missing previous time")
	}
	if !strings.Contains(msg.HTML, "تغییر کرد") {
		t.Fatalf("html missing reschedule copy")
	}
}

func TestBookingCancelledRecruiterMessage_CandidateCancelled(t *testing.T) {
	msg := email.BookingCancelledRecruiterMessage(email.BookingNotificationData{
		CandidateName:      "Ali",
		CandidateEmail:     "ali@example.com",
		CandidatePhone:     "+989121234567",
		RecruiterEmail:     "recruiter@example.com",
		InterviewTypeTitle: "Backend Interview",
		SlotStartAt:        time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		SlotEndAt:          time.Date(2026, 7, 7, 11, 0, 0, 0, time.UTC),
		CancelledBy:        "candidate",
	})

	if msg.To != "recruiter@example.com" {
		t.Fatalf("to = %q", msg.To)
	}
	if !strings.Contains(msg.HTML, "رزرو خود را لغو کرد") {
		t.Fatalf("html = %s", msg.HTML)
	}
}

func TestBookingCreatedRecruiterMessage_IncludesCandidateDetails(t *testing.T) {
	msg := email.BookingCreatedRecruiterMessage(email.BookingNotificationData{
		CandidateName:      "Ali",
		CandidateEmail:     "ali@example.com",
		CandidatePhone:     "+989121234567",
		RecruiterEmail:     "recruiter@example.com",
		OrganizationName:   "Acme",
		InterviewTypeTitle: "Backend Interview",
		SlotStartAt:        time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
		SlotEndAt:          time.Date(2026, 7, 7, 11, 0, 0, 0, time.UTC),
		MeetLink:           "https://meet.google.com/abc-defg-hij",
	})

	if msg.To != "recruiter@example.com" {
		t.Fatalf("to = %q", msg.To)
	}
	if !strings.Contains(msg.Subject, "رزرو جدید") {
		t.Fatalf("subject = %q", msg.Subject)
	}
	if !strings.Contains(msg.HTML, "ali@example.com") {
		t.Fatalf("html missing candidate email")
	}
}

func TestBookingReminder24hMessage_IncludesActions(t *testing.T) {
	msg := email.BookingReminder24hMessage(email.BookingNotificationData{
		CandidateName:      "Ali",
		CandidateEmail:     "ali@example.com",
		OrganizationName:   "Acme",
		InterviewTypeTitle: "Backend Interview",
		SlotStartAt:        time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
		SlotEndAt:          time.Date(2026, 7, 8, 11, 0, 0, 0, time.UTC),
		RescheduleURL:      "http://localhost:3000/reschedule/token",
		CancelURL:          "http://localhost:3000/cancel/token",
	})

	if msg.To != "ali@example.com" {
		t.Fatalf("to = %q", msg.To)
	}
	if !strings.Contains(msg.HTML, "یادآوری") {
		t.Fatalf("html missing reminder copy")
	}
	if !strings.Contains(msg.HTML, "/reschedule/token") {
		t.Fatalf("html missing reschedule link")
	}
}
