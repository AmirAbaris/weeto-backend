package notification_test

import (
	"context"
	"strings"
	"testing"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/platform/email"
	"github.com/AmirAbaris/weeto-backend/internal/worker/notification"
)

type recordingSender struct {
	messages []email.Message
}

func (s *recordingSender) Send(ctx context.Context, msg email.Message) error {
	s.messages = append(s.messages, msg)
	return nil
}

func TestProcessor_SendBookingConfirmationToCandidate(t *testing.T) {
	sender := &recordingSender{}
	processor := notification.NewProcessor(nil, nil, sender, "http://localhost:3000")

	payload := []byte(`{
		"recipient": "candidate",
		"candidate_name": "Ali",
		"candidate_email": "ali@example.com",
		"organization_name": "Acme",
		"interview_type_title": "Backend Interview",
		"slot_start_at": "2026-07-07T10:00:00Z",
		"slot_end_at": "2026-07-07T11:00:00Z",
		"reschedule_token": "reschedule-token",
		"cancel_token": "cancel-token"
	}`)

	row := db.NotificationOutbox{
		EventType: db.NotificationEventTypeBookingCreated,
		Payload:   payload,
	}

	if err := processor.ProcessRow(context.Background(), row); err != nil {
		t.Fatalf("process row: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(sender.messages))
	}
	msg := sender.messages[0]
	if msg.To != "ali@example.com" {
		t.Fatalf("to = %q", msg.To)
	}
	if !strings.Contains(msg.HTML, "/reschedule/reschedule-token") {
		t.Fatalf("html missing reschedule link")
	}
	if !strings.Contains(msg.HTML, "/cancel/cancel-token") {
		t.Fatalf("html missing cancel link")
	}
}

func TestProcessor_SendBookingCreatedToRecruiter(t *testing.T) {
	sender := &recordingSender{}
	processor := notification.NewProcessor(nil, nil, sender, "http://localhost:3000")

	payload := []byte(`{
		"recipient": "recruiter",
		"candidate_name": "Ali",
		"candidate_email": "ali@example.com",
		"candidate_phone": "+989121234567",
		"recruiter_email": "recruiter@example.com",
		"organization_name": "Acme",
		"interview_type_title": "Backend Interview",
		"slot_start_at": "2026-07-07T10:00:00Z",
		"slot_end_at": "2026-07-07T11:00:00Z"
	}`)

	row := db.NotificationOutbox{
		EventType: db.NotificationEventTypeBookingCreated,
		Payload:   payload,
	}

	if err := processor.ProcessRow(context.Background(), row); err != nil {
		t.Fatalf("process row: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(sender.messages))
	}
	if sender.messages[0].To != "recruiter@example.com" {
		t.Fatalf("to = %q", sender.messages[0].To)
	}
	if !strings.Contains(sender.messages[0].Subject, "رزرو جدید") {
		t.Fatalf("unexpected recruiter booking subject: %q", sender.messages[0].Subject)
	}
	if !strings.Contains(sender.messages[0].HTML, "رزرو کرد") {
		t.Fatalf("unexpected recruiter booking html: %s", sender.messages[0].HTML)
	}
}

func TestProcessor_SendReminder24hToCandidate(t *testing.T) {
	sender := &recordingSender{}
	processor := notification.NewProcessor(nil, nil, sender, "http://localhost:3000")

	payload := []byte(`{
		"recipient": "candidate",
		"candidate_name": "Ali",
		"candidate_email": "ali@example.com",
		"organization_name": "Acme",
		"interview_type_title": "Backend Interview",
		"slot_start_at": "2026-07-08T10:00:00Z",
		"slot_end_at": "2026-07-08T11:00:00Z",
		"reschedule_token": "reschedule-token",
		"cancel_token": "cancel-token"
	}`)

	row := db.NotificationOutbox{
		EventType: db.NotificationEventTypeReminder24h,
		Payload:   payload,
	}

	if err := processor.ProcessRow(context.Background(), row); err != nil {
		t.Fatalf("process row: %v", err)
	}
	if sender.messages[0].To != "ali@example.com" {
		t.Fatalf("to = %q", sender.messages[0].To)
	}
	if !strings.Contains(sender.messages[0].HTML, "یادآوری") {
		t.Fatalf("html missing reminder copy")
	}
}

func TestProcessor_SendBookingRescheduledToCandidate(t *testing.T) {
	sender := &recordingSender{}
	processor := notification.NewProcessor(nil, nil, sender, "http://localhost:3000")

	payload := []byte(`{
		"recipient": "candidate",
		"candidate_name": "Ali",
		"candidate_email": "ali@example.com",
		"organization_name": "Acme",
		"interview_type_title": "Backend Interview",
		"slot_start_at": "2026-07-07T11:00:00Z",
		"slot_end_at": "2026-07-07T12:00:00Z",
		"previous_slot_start_at": "2026-07-07T10:00:00Z",
		"previous_slot_end_at": "2026-07-07T11:00:00Z",
		"reschedule_token": "reschedule-token",
		"cancel_token": "cancel-token"
	}`)

	row := db.NotificationOutbox{
		EventType: db.NotificationEventTypeBookingRescheduled,
		Payload:   payload,
	}

	if err := processor.ProcessRow(context.Background(), row); err != nil {
		t.Fatalf("process row: %v", err)
	}
	if sender.messages[0].To != "ali@example.com" {
		t.Fatalf("to = %q", sender.messages[0].To)
	}
	if !strings.Contains(sender.messages[0].HTML, "تغییر کرد") {
		t.Fatalf("html missing reschedule copy")
	}
}

func TestProcessor_SendBookingCancelled(t *testing.T) {
	sender := &recordingSender{}
	processor := notification.NewProcessor(nil, nil, sender, "http://localhost:3000")

	for _, tc := range []struct {
		recipient string
		to        string
		payload   string
	}{
		{
			recipient: "candidate",
			to:        "ali@example.com",
			payload: `{
				"recipient": "candidate",
				"candidate_name": "Ali",
				"candidate_email": "ali@example.com",
				"recruiter_email": "recruiter@example.com",
				"organization_name": "Acme",
				"interview_type_title": "Backend Interview",
				"slot_start_at": "2026-07-07T10:00:00Z",
				"slot_end_at": "2026-07-07T11:00:00Z"
			}`,
		},
		{
			recipient: "recruiter",
			to:        "recruiter@example.com",
			payload: `{
				"recipient": "recruiter",
				"candidate_name": "Ali",
				"candidate_email": "ali@example.com",
				"candidate_phone": "+989121234567",
				"recruiter_email": "recruiter@example.com",
				"organization_name": "Acme",
				"interview_type_title": "Backend Interview",
				"slot_start_at": "2026-07-07T10:00:00Z",
				"slot_end_at": "2026-07-07T11:00:00Z",
				"cancelled_by": "candidate"
			}`,
		},
	} {
		sender.messages = nil
		row := db.NotificationOutbox{
			EventType: db.NotificationEventTypeBookingCancelled,
			Payload:   []byte(tc.payload),
		}
		if err := processor.ProcessRow(context.Background(), row); err != nil {
			t.Fatalf("process row %s: %v", tc.recipient, err)
		}
		if len(sender.messages) != 1 || sender.messages[0].To != tc.to {
			t.Fatalf("%s: got %+v, want to %s", tc.recipient, sender.messages, tc.to)
		}
	}
}
