package google

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type EventInput struct {
	Summary        string
	CandidateEmail string
	StartAt        time.Time
	EndAt          time.Time
}

type EventResult struct {
	EventID  string
	MeetLink string
}

type CalendarClient interface {
	CreateInterviewEvent(ctx context.Context, ownerID pgtype.UUID, in EventInput) (EventResult, error)
	UpdateInterviewEvent(ctx context.Context, ownerID pgtype.UUID, eventID string, in EventInput) error
	DeleteEvent(ctx context.Context, ownerID pgtype.UUID, eventID string) error
}
