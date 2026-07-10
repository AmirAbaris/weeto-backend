package google

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type NoopCalendar struct{}

func (n *NoopCalendar) CreateInterviewEvent(ctx context.Context, ownerID pgtype.UUID, in EventInput) (EventResult, error) {
	return EventResult{}, ErrNotConnected
}

func (n *NoopCalendar) UpdateInterviewEvent(ctx context.Context, ownerID pgtype.UUID, eventID string, in EventInput) error {
	return ErrNotConnected
}

func (n *NoopCalendar) DeleteEvent(ctx context.Context, ownerID pgtype.UUID, eventID string) error {
	return ErrNotConnected
}
