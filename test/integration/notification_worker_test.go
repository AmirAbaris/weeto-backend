package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	smsplatform "github.com/AmirAbaris/weeto-backend/internal/platform/sms"
	notificationsvc "github.com/AmirAbaris/weeto-backend/internal/service/notification"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
)

type recordingSender struct {
	calls []smsplatform.Parameter
}

func (r *recordingSender) VerifySend(_ context.Context, mobile string, templateID int, params []smsplatform.Parameter) (int64, float64, error) {
	_ = mobile
	_ = templateID
	if len(params) > 0 {
		r.calls = append(r.calls, params[0])
	}
	return 1, 1, nil
}

func TestNotificationWorkerProcessesBookingCreated(t *testing.T) {
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	_, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Ali Rezaei",
		Phone:  "09121234567",
		Email:  "ali@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	sender := &recordingSender{}
	cfg := &config.Config{SMSAPIKey: "test-key", SMSTemplateID: 123456}
	svc := notificationsvc.NewService(env.Pool, env.Queries, sender, cfg)

	n, err := svc.ProcessPending(env.Ctx, 10)
	if err != nil {
		t.Fatalf("process pending: %v", err)
	}
	if n != 2 {
		t.Fatalf("processed = %d, want 2", n)
	}
	if len(sender.calls) != 1 {
		t.Fatalf("sms calls = %d, want 1", len(sender.calls))
	}
	if sender.calls[0].Name != "Code" {
		t.Fatalf("param name = %q", sender.calls[0].Name)
	}

	outbox, err := env.Queries.ListNotificationOutboxByOrg(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	for _, row := range outbox {
		if row.Status != db.NotificationStatusSent {
			t.Fatalf("status = %q, want sent", row.Status)
		}
	}

	var candidatePayload map[string]any
	for _, row := range outbox {
		var payload map[string]any
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload["recipient"] == "candidate" {
			candidatePayload = payload
		}
	}
	if candidatePayload == nil {
		t.Fatal("candidate outbox row not found")
	}
}
