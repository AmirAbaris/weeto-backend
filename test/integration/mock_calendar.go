package integration

import (
	"context"
	"errors"
	"sync"

	googleplatform "github.com/AmirAbaris/weeto-backend/internal/platform/google"
	"github.com/jackc/pgx/v5/pgtype"
)

type MockCalendarClient struct {
	mu sync.Mutex

	CreateFn func(ctx context.Context, ownerID pgtype.UUID, in googleplatform.EventInput) (googleplatform.EventResult, error)
	UpdateFn func(ctx context.Context, ownerID pgtype.UUID, eventID string, in googleplatform.EventInput) error
	DeleteFn func(ctx context.Context, ownerID pgtype.UUID, eventID string) error

	Created []googleplatform.EventInput
	Deleted []string
}

func (m *MockCalendarClient) CreateInterviewEvent(ctx context.Context, ownerID pgtype.UUID, in googleplatform.EventInput) (googleplatform.EventResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Created = append(m.Created, in)
	if m.CreateFn != nil {
		return m.CreateFn(ctx, ownerID, in)
	}
	return googleplatform.EventResult{
		EventID:  "mock-event-id",
		MeetLink: "https://meet.google.com/mock-link",
	}, nil
}

func (m *MockCalendarClient) UpdateInterviewEvent(ctx context.Context, ownerID pgtype.UUID, eventID string, in googleplatform.EventInput) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, ownerID, eventID, in)
	}
	return nil
}

func (m *MockCalendarClient) DeleteEvent(ctx context.Context, ownerID pgtype.UUID, eventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Deleted = append(m.Deleted, eventID)
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, ownerID, eventID)
	}
	return nil
}

func calendarCreateError() func(context.Context, pgtype.UUID, googleplatform.EventInput) (googleplatform.EventResult, error) {
	return func(context.Context, pgtype.UUID, googleplatform.EventInput) (googleplatform.EventResult, error) {
		return googleplatform.EventResult{}, errors.New("calendar unavailable")
	}
}
