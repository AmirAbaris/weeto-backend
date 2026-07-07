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
