package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/platform/email"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxRetries = 5

type Processor struct {
	pool        *pgxpool.Pool
	q           *db.Queries
	sender      email.Sender
	frontendURL string
}

func NewProcessor(pool *pgxpool.Pool, q *db.Queries, sender email.Sender, frontendURL string) *Processor {
	return &Processor{
		pool:        pool,
		q:           q,
		sender:      sender,
		frontendURL: frontendURL,
	}
}

func (p *Processor) ProcessBatch(ctx context.Context, batchSize int32) (int, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	qtx := p.q.WithTx(tx)

	rows, err := qtx.ClaimPendingNotifications(ctx, batchSize)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}

	processed := 0
	for _, row := range rows {
		if err := p.processRow(ctx, qtx, row); err != nil {
			slog.Error("notification process failed",
				"outbox_id", row.ID,
				"event_type", row.EventType,
				"err", err,
			)
			if _, markErr := qtx.MarkNotificationFailed(ctx, db.MarkNotificationFailedParams{
				ID:         row.ID,
				RetryCount: maxRetries,
			}); markErr != nil {
				return processed, fmt.Errorf("mark failed for %s: %w", row.ID, markErr)
			}
			continue
		}

		if _, err := qtx.MarkNotificationSent(ctx, row.ID); err != nil {
			return processed, err
		}
		processed++
	}

	if err := tx.Commit(ctx); err != nil {
		return processed, err
	}

	return processed, nil
}

func (p *Processor) ProcessRow(ctx context.Context, row db.NotificationOutbox) error {
	return p.processRow(ctx, nil, row)
}

func (p *Processor) processRow(ctx context.Context, qtx *db.Queries, row db.NotificationOutbox) error {
	switch row.EventType {
	case db.NotificationEventTypeBookingCreated:
		return p.sendBookingConfirmation(ctx, row.Payload)
	default:
		return fmt.Errorf("unsupported event type: %s", row.EventType)
	}
}

func (p *Processor) sendBookingConfirmation(ctx context.Context, payload []byte) error {
	var meta struct {
		Recipient string `json:"recipient"`
	}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return err
	}
	if meta.Recipient != "candidate" {
		return fmt.Errorf("unexpected recipient: %s", meta.Recipient)
	}

	data, err := email.ParseBookingConfirmationPayload(payload, p.frontendURL)
	if err != nil {
		return err
	}
	if data.CandidateEmail == "" {
		return fmt.Errorf("candidate_email is required")
	}

	msg := email.BookingConfirmationMessage(data)
	return p.sender.Send(ctx, msg)
}
