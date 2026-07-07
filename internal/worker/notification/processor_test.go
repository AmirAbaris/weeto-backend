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
