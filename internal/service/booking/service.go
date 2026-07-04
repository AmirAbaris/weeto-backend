package booking

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	slotsvc "github.com/AmirAbaris/weeto-backend/internal/service/slot"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool    *pgxpool.Pool
	q       *db.Queries
	orgSvc  *orgsvc.Service
	slotSvc *slotsvc.Service
}

func NewService(pool *pgxpool.Pool, q *db.Queries, orgSvc *orgsvc.Service, slotSvc *slotsvc.Service) *Service {
	return &Service{pool: pool, q: q, orgSvc: orgSvc, slotSvc: slotSvc}
}

type BookInput struct {
	SlotID pgtype.UUID
	Name   string
	Phone  string
	Email  string
}

type Metadata struct {
	Organization  db.Organization
	InterviewType db.InterviewType
}

type SlotView struct {
	ID      string    `json:"id"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

type BookingResult struct {
	Booking db.Booking
	Slot    db.Slot
}

func (s *Service) ResolveType(ctx context.Context, orgSlug, typeSlug string) (Metadata, error) {
	org, err := s.orgSvc.GetBySlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, orgsvc.ErrOrgNotFound) {
			return Metadata{}, ErrOrgNotFound
		}
		return Metadata{}, err
	}

	typeSlug = normalizeSlug(typeSlug)
	it, err := s.q.GetInterviewTypeByOrgAndSlug(ctx, db.GetInterviewTypeByOrgAndSlugParams{
		OrganizationID: org.ID,
		Slug:           typeSlug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Metadata{}, ErrTypeNotFound
		}
		return Metadata{}, err
	}

	return Metadata{Organization: org, InterviewType: it}, nil
}

func (s *Service) ListAvailableSlots(ctx context.Context, orgSlug, typeSlug string) ([]SlotView, error) {
	meta, err := s.ResolveType(ctx, orgSlug, typeSlug)
	if err != nil {
		return nil, err
	}

	windowStart, windowEnd, err := s.slotWindow(ctx, meta.Organization.ID)
	if err != nil {
		return nil, err
	}

	slots, err := s.q.ListAvailableSlotsByType(ctx, db.ListAvailableSlotsByTypeParams{
		OrganizationID:  meta.Organization.ID,
		InterviewTypeID: meta.InterviewType.ID,
		StartAt:         timestamptz(windowStart),
		StartAt_2:       timestamptz(windowEnd),
	})
	if err != nil {
		return nil, err
	}

	out := make([]SlotView, 0, len(slots))
	for _, slot := range slots {
		out = append(out, toSlotView(slot))
	}
	return out, nil
}

func (s *Service) Book(ctx context.Context, orgSlug, typeSlug string, in BookInput) (BookingResult, error) {
	if !in.SlotID.Valid {
		return BookingResult{}, ErrInvalidSlotID
	}

	name, phone, email, err := validateCandidate(in.Name, in.Phone, in.Email)
	if err != nil {
		return BookingResult{}, err
	}

	meta, err := s.ResolveType(ctx, orgSlug, typeSlug)
	if err != nil {
		return BookingResult{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BookingResult{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	slot, err := qtx.GetSlotForUpdate(ctx, db.GetSlotForUpdateParams{
		ID:              in.SlotID,
		OrganizationID:  meta.Organization.ID,
		InterviewTypeID: meta.InterviewType.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BookingResult{}, ErrSlotUnavailable
		}
		return BookingResult{}, err
	}

	rescheduleToken, err := randomToken()
	if err != nil {
		return BookingResult{}, err
	}
	cancelToken, err := randomToken()
	if err != nil {
		return BookingResult{}, err
	}

	booking, err := qtx.InsertBooking(ctx, db.InsertBookingParams{
		OrganizationID:  meta.Organization.ID,
		InterviewTypeID: meta.InterviewType.ID,
		SlotID:          slot.ID,
		CandidateName:   name,
		CandidatePhone:  phone,
		CandidateEmail:  email,
		RescheduleToken: rescheduleToken,
		CancelToken:     cancelToken,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return BookingResult{}, ErrSlotUnavailable
		}
		return BookingResult{}, err
	}

	rows, err := qtx.MarkSlotBooked(ctx, slot.ID)
	if err != nil {
		return BookingResult{}, err
	}
	if rows == 0 {
		return BookingResult{}, ErrSlotUnavailable
	}

	payload, err := bookingPayload(meta.Organization, meta.InterviewType, booking, slot)
	if err != nil {
		return BookingResult{}, err
	}

	if _, err := qtx.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
		OrganizationID: meta.Organization.ID,
		EventType:      db.NotificationEventTypeBookingCreated,
		Payload:        payload,
	}); err != nil {
		return BookingResult{}, err
	}

	if _, err := qtx.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
		OrganizationID: meta.Organization.ID,
		EventType:      db.NotificationEventTypeBookingCreated,
		Payload:        payload,
	}); err != nil {
		return BookingResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return BookingResult{}, err
	}

	return BookingResult{Booking: booking, Slot: slot}, nil
}

func (s *Service) slotWindow(ctx context.Context, orgID pgtype.UUID) (time.Time, time.Time, error) {
	loc := defaultLocation()
	settings, err := s.q.GetAvailabilitySettingsByOrg(ctx, orgID)
	if err == nil && settings.Timezone != "" {
		if parsed, err := time.LoadLocation(settings.Timezone); err == nil {
			loc = parsed
		}
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, time.Time{}, err
	}

	now := s.slotSvc.Now()
	todayStart := startOfDay(now, loc)
	start := maxTime(now.UTC(), todayStart)
	end := todayStart.AddDate(0, 0, slotsvc.DefaultWindowDays)
	return start, end, nil
}

func bookingPayload(org db.Organization, it db.InterviewType, booking db.Booking, slot db.Slot) ([]byte, error) {
	data := map[string]any{
		"booking_id":            booking.ID.String(),
		"organization_name":     org.Name,
		"organization_slug":     org.Slug,
		"interview_type_title":  it.Title,
		"interview_type_slug":   it.Slug,
		"candidate_name":        booking.CandidateName,
		"candidate_phone":       booking.CandidatePhone,
		"candidate_email":       booking.CandidateEmail,
		"slot_start_at":         slot.StartAt.Time.UTC().Format(time.RFC3339),
		"slot_end_at":           slot.EndAt.Time.UTC().Format(time.RFC3339),
		"reschedule_token":      booking.RescheduleToken,
		"cancel_token":          booking.CancelToken,
	}
	return json.Marshal(data)
}

func toSlotView(slot db.Slot) SlotView {
	return SlotView{
		ID:      slot.ID.String(),
		StartAt: slot.StartAt.Time.UTC(),
		EndAt:   slot.EndAt.Time.UTC(),
	}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func defaultLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		return time.UTC
	}
	return loc
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
