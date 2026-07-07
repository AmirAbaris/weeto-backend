package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/platform/email"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	"github.com/AmirAbaris/weeto-backend/internal/worker/notification"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
)

func TestWorkerProcessesBookingCreatedEmails(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	_, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Ali",
		Phone:  "+989121234567",
		Email:  "ali@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	processor := notification.NewProcessor(env.Pool, env.Queries, email.NewNoopSender(), "http://localhost:3000")
	processed, err := processor.ProcessBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("process batch: %v", err)
	}
	if processed != 2 {
		t.Fatalf("processed = %d, want 2", processed)
	}

	outbox, err := env.Queries.ListNotificationOutboxByOrg(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}

	var candidateSent, recruiterSent int
	for _, row := range outbox {
		if row.EventType != db.NotificationEventTypeBookingCreated {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		recipient, _ := payload["recipient"].(string)
		if row.Status != db.NotificationStatusSent {
			t.Fatalf("%s row status = %q, want sent", recipient, row.Status)
		}
		switch recipient {
		case "candidate":
			candidateSent++
		case "recruiter":
			recruiterSent++
		}
	}
	if candidateSent != 1 || recruiterSent != 1 {
		t.Fatalf("candidateSent=%d recruiterSent=%d, want 1/1", candidateSent, recruiterSent)
	}
}
