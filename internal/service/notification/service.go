package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	smsplatform "github.com/AmirAbaris/weeto-backend/internal/platform/sms"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	sender smsplatform.Sender
	cfg    *config.Config
}

func NewService(pool *pgxpool.Pool, q *db.Queries, sender smsplatform.Sender, cfg *config.Config) *Service {
	return &Service{pool: pool, q: q, sender: sender, cfg: cfg}
}

type bookingPayload struct {
	BookingID      string `json:"booking_id"`
	CandidatePhone string `json:"candidate_phone"`
	RescheduleToken string `json:"reschedule_token"`
	Recipient      string `json:"recipient"`
}

func (s *Service) ProcessPending(ctx context.Context, limit int32) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	rows, err := qtx.ListPendingNotificationOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, row := range rows {
		if err := s.processRow(ctx, qtx, row); err != nil {
			slog.Error("notification process failed",
				"outbox_id", row.ID,
				"event_type", row.EventType,
				"err", err,
			)
			if markErr := qtx.MarkNotificationOutboxFailed(ctx, row.ID); markErr != nil {
				return processed, markErr
			}
			continue
		}
		if err := qtx.MarkNotificationOutboxSent(ctx, row.ID); err != nil {
			return processed, err
		}
		processed++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return processed, nil
}

func (s *Service) processRow(ctx context.Context, qtx *db.Queries, row db.NotificationOutbox) error {
	switch row.EventType {
	case db.NotificationEventTypeBookingCreated:
		return s.processBookingCreated(ctx, row.Payload)
	default:
		// recruiter email and other events are not implemented yet
		return nil
	}
}

func (s *Service) processBookingCreated(ctx context.Context, raw []byte) error {
	var payload bookingPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if payload.Recipient != "candidate" {
		return nil
	}
	if s.cfg.SMSAPIKey == "" {
		return nil
	}
	if s.cfg.SMSTemplateID == 0 {
		return fmt.Errorf("SMS_TEMPLATE_ID is required when SMS_API_KEY is set")
	}

	mobile, err := smsplatform.NormalizeMobile(payload.CandidatePhone)
	if err != nil {
		return fmt.Errorf("normalize phone: %w", err)
	}

	code := bookingCode(payload)
	_, _, err = s.sender.VerifySend(ctx, mobile, s.cfg.SMSTemplateID, []smsplatform.Parameter{
		{Name: "Code", Value: code},
	})
	return err
}

func bookingCode(payload bookingPayload) string {
	token := strings.TrimSpace(payload.RescheduleToken)
	if len(token) >= 6 {
		return token[:6]
	}
	id := strings.ReplaceAll(payload.BookingID, "-", "")
	if len(id) >= 6 {
		return id[:6]
	}
	return "000000"
}
