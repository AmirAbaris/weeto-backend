package google

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type calendarClient struct {
	cfg    *config.Config
	tokens *tokenManager
}

func NewCalendarClient(cfg *config.Config, q *db.Queries) CalendarClient {
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return &NoopCalendar{}
	}
	return &calendarClient{
		cfg:    cfg,
		tokens: newTokenManager(cfg, q),
	}
}

func (c *calendarClient) CreateInterviewEvent(ctx context.Context, ownerID pgtype.UUID, in EventInput) (EventResult, error) {
	result, err := c.insertEvent(ctx, ownerID, in)
	if err == nil {
		return result, nil
	}
	if !isUnauthorized(err) {
		return EventResult{}, err
	}
	return c.insertEvent(ctx, ownerID, in)
}

func (c *calendarClient) UpdateInterviewEvent(ctx context.Context, ownerID pgtype.UUID, eventID string, in EventInput) error {
	err := c.patchEvent(ctx, ownerID, eventID, in)
	if err == nil || !isUnauthorized(err) {
		return err
	}
	return c.patchEvent(ctx, ownerID, eventID, in)
}

func (c *calendarClient) DeleteEvent(ctx context.Context, ownerID pgtype.UUID, eventID string) error {
	err := c.deleteEvent(ctx, ownerID, eventID)
	if err == nil || !isUnauthorized(err) {
		return err
	}
	return c.deleteEvent(ctx, ownerID, eventID)
}

func (c *calendarClient) insertEvent(ctx context.Context, ownerID pgtype.UUID, in EventInput) (EventResult, error) {
	svc, err := c.calendarService(ctx, ownerID)
	if err != nil {
		return EventResult{}, err
	}

	requestID, err := randomRequestID()
	if err != nil {
		return EventResult{}, err
	}

	event := &calendar.Event{
		Summary: in.Summary,
		Start: &calendar.EventDateTime{
			DateTime: in.StartAt.UTC().Format(time.RFC3339),
			TimeZone: "UTC",
		},
		End: &calendar.EventDateTime{
			DateTime: in.EndAt.UTC().Format(time.RFC3339),
			TimeZone: "UTC",
		},
		ConferenceData: &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId: requestID,
				ConferenceSolutionKey: &calendar.ConferenceSolutionKey{
					Type: "hangoutsMeet",
				},
			},
		},
	}
	if in.CandidateEmail != "" {
		event.Attendees = []*calendar.EventAttendee{{Email: in.CandidateEmail}}
	}

	created, err := svc.Events.Insert("primary", event).ConferenceDataVersion(1).SendUpdates("none").Context(ctx).Do()
	if err != nil {
		return EventResult{}, wrapCalendarError(err)
	}

	meetLink := meetLinkFromEvent(created)
	if meetLink == "" {
		return EventResult{}, fmt.Errorf("%w: meet link missing from created event", ErrCalendarAPI)
	}

	return EventResult{
		EventID:  created.Id,
		MeetLink: meetLink,
	}, nil
}

func (c *calendarClient) patchEvent(ctx context.Context, ownerID pgtype.UUID, eventID string, in EventInput) error {
	svc, err := c.calendarService(ctx, ownerID)
	if err != nil {
		return err
	}

	event := &calendar.Event{
		Summary: in.Summary,
		Start: &calendar.EventDateTime{
			DateTime: in.StartAt.UTC().Format(time.RFC3339),
			TimeZone: "UTC",
		},
		End: &calendar.EventDateTime{
			DateTime: in.EndAt.UTC().Format(time.RFC3339),
			TimeZone: "UTC",
		},
	}

	_, err = svc.Events.Patch("primary", eventID, event).SendUpdates("none").Context(ctx).Do()
	if err != nil {
		return wrapCalendarError(err)
	}
	return nil
}

func (c *calendarClient) deleteEvent(ctx context.Context, ownerID pgtype.UUID, eventID string) error {
	svc, err := c.calendarService(ctx, ownerID)
	if err != nil {
		return err
	}

	if err := svc.Events.Delete("primary", eventID).SendUpdates("none").Context(ctx).Do(); err != nil {
		return wrapCalendarError(err)
	}
	return nil
}

func (c *calendarClient) calendarService(ctx context.Context, ownerID pgtype.UUID) (*calendar.Service, error) {
	src, err := c.tokens.oauth2TokenSource(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	return calendar.NewService(ctx, option.WithTokenSource(src))
}

func meetLinkFromEvent(event *calendar.Event) string {
	if event.HangoutLink != "" {
		return event.HangoutLink
	}
	if event.ConferenceData != nil {
		for _, ep := range event.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" && ep.Uri != "" {
				return ep.Uri
			}
		}
	}
	return ""
}

func randomRequestID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isUnauthorized(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusUnauthorized
	}
	return false
}

func wrapCalendarError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotConnected) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrCalendarAPI, err)
}
