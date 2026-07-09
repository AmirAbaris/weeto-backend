package booking

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	googleplatform "github.com/AmirAbaris/weeto-backend/internal/platform/google"
	"github.com/AmirAbaris/weeto-backend/internal/plan"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	slotsvc "github.com/AmirAbaris/weeto-backend/internal/service/slot"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool     *pgxpool.Pool
	q        *db.Queries
	orgSvc   *orgsvc.Service
	slotSvc  *slotsvc.Service
	calendar googleplatform.CalendarClient
}

func NewService(pool *pgxpool.Pool, q *db.Queries, orgSvc *orgsvc.Service, slotSvc *slotsvc.Service, calendar googleplatform.CalendarClient) *Service {
	if calendar == nil {
		calendar = &googleplatform.NoopCalendar{}
	}
	return &Service{pool: pool, q: q, orgSvc: orgSvc, slotSvc: slotSvc, calendar: calendar}
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

type BookingView struct {
	ID             string    `json:"id"`
	SlotID         string    `json:"slot_id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	Status         string    `json:"status"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	InterviewTitle string    `json:"interview_title"`
}

type ListResult struct {
	Today    []BookingView `json:"today"`
	Upcoming []BookingView `json:"upcoming"`
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

	isGoogleMeet := meta.InterviewType.MeetingProvider == db.MeetingProviderGoogleMeet
	if isGoogleMeet {
		if err := s.ensureGoogleConnected(ctx, meta.Organization.OwnerID); err != nil {
			return BookingResult{}, err
		}
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

	payloadCandidate, err := notificationPayload(ctx, qtx, meta.Organization, meta.InterviewType, booking, slot, "", "candidate", nil)
	if err != nil {
		return BookingResult{}, err
	}
	payloadRecruiter, err := notificationPayload(ctx, qtx, meta.Organization, meta.InterviewType, booking, slot, "", "recruiter", nil)
	if err != nil {
		return BookingResult{}, err
	}

	if !isGoogleMeet {
		if _, err := qtx.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
			OrganizationID: meta.Organization.ID,
			EventType:      db.NotificationEventTypeBookingCreated,
			Payload:        payloadCandidate,
		}); err != nil {
			return BookingResult{}, err
		}

		if _, err := qtx.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
			OrganizationID: meta.Organization.ID,
			EventType:      db.NotificationEventTypeBookingCreated,
			Payload:        payloadRecruiter,
		}); err != nil {
			return BookingResult{}, err
		}

		if err := insertReminderNotification(ctx, qtx, meta.Organization, meta.InterviewType, booking, slot, "", time.Now().UTC()); err != nil {
			return BookingResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return BookingResult{}, err
	}

	if isGoogleMeet {
		booking, err = s.finalizeGoogleMeetBooking(ctx, meta, booking, slot)
		if err != nil {
			return BookingResult{}, err
		}
	}

	return BookingResult{Booking: booking, Slot: slot}, nil
}

func (s *Service) finalizeGoogleMeetBooking(ctx context.Context, meta Metadata, booking db.Booking, slot db.Slot) (db.Booking, error) {
	if err := s.q.ResetMeetLinksPeriodIfNeeded(ctx, meta.Organization.ID); err != nil {
		s.compensatingCancel(ctx, meta.Organization, booking)
		return db.Booking{}, err
	}

	if _, err := s.q.TryIncrementMeetLinksUsed(ctx, db.TryIncrementMeetLinksUsedParams{
		ID:            meta.Organization.ID,
		MeetLinksUsed: plan.MeetLinksLimitForIncrement(meta.Organization.Plan),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.compensatingCancel(ctx, meta.Organization, booking)
			return db.Booking{}, ErrMeetLinkLimitReached
		}
		s.compensatingCancel(ctx, meta.Organization, booking)
		return db.Booking{}, err
	}

	event, err := s.calendar.CreateInterviewEvent(ctx, meta.Organization.OwnerID, googleplatform.EventInput{
		Summary:        fmt.Sprintf("%s — %s", meta.InterviewType.Title, booking.CandidateName),
		CandidateEmail: booking.CandidateEmail,
		StartAt:        slot.StartAt.Time,
		EndAt:          slot.EndAt.Time,
	})
	if err != nil {
		_ = s.q.DecrementMeetLinksUsed(ctx, meta.Organization.ID)
		s.compensatingCancel(ctx, meta.Organization, booking)
		if errors.Is(err, googleplatform.ErrNotConnected) {
			return db.Booking{}, ErrGoogleNotConnected
		}
		slog.Error("google calendar create failed",
			"org_id", meta.Organization.ID,
			"booking_id", booking.ID,
			"err", err,
		)
		return db.Booking{}, ErrGoogleCalendarFailed
	}

	updated, err := s.q.UpdateBookingMeetInfo(ctx, db.UpdateBookingMeetInfoParams{
		ID:              booking.ID,
		MeetLink:        pgtype.Text{String: event.MeetLink, Valid: true},
		CalendarEventID: pgtype.Text{String: event.EventID, Valid: true},
	})
	if err != nil {
		_ = s.q.DecrementMeetLinksUsed(ctx, meta.Organization.ID)
		_ = s.calendar.DeleteEvent(ctx, meta.Organization.OwnerID, event.EventID)
		s.compensatingCancel(ctx, meta.Organization, booking)
		return db.Booking{}, ErrGoogleCalendarFailed
	}

	payloadCandidate, err := notificationPayload(ctx, s.q, meta.Organization, meta.InterviewType, updated, slot, event.MeetLink, "candidate", nil)
	if err != nil {
		return updated, err
	}
	payloadRecruiter, err := notificationPayload(ctx, s.q, meta.Organization, meta.InterviewType, updated, slot, event.MeetLink, "recruiter", nil)
	if err != nil {
		return updated, err
	}

	for _, payload := range [][]byte{payloadCandidate, payloadRecruiter} {
		if _, err := s.q.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
			OrganizationID: meta.Organization.ID,
			EventType:      db.NotificationEventTypeBookingCreated,
			Payload:        payload,
		}); err != nil {
			return updated, err
		}
	}

	if err := insertReminderNotification(ctx, s.q, meta.Organization, meta.InterviewType, updated, slot, event.MeetLink, time.Now().UTC()); err != nil {
		return updated, err
	}

	return updated, nil
}

func (s *Service) compensatingCancel(ctx context.Context, org db.Organization, booking db.Booking) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("compensating cancel begin failed", "booking_id", booking.ID, "err", err)
		return
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	if _, err := qtx.CancelBooking(ctx, db.CancelBookingParams{
		ID:             booking.ID,
		OrganizationID: org.ID,
	}); err != nil {
		slog.Error("compensating cancel booking failed", "booking_id", booking.ID, "err", err)
		return
	}

	if err := qtx.SetSlotBooked(ctx, db.SetSlotBookedParams{
		ID:     booking.SlotID,
		Booked: false,
	}); err != nil {
		slog.Error("compensating free slot failed", "booking_id", booking.ID, "err", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("compensating cancel commit failed", "booking_id", booking.ID, "err", err)
	}
}

func (s *Service) ensureGoogleConnected(ctx context.Context, ownerID pgtype.UUID) error {
	connected, err := s.q.IsGoogleConnected(ctx, ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGoogleNotConnected
		}
		return err
	}
	if !connected {
		return ErrGoogleNotConnected
	}
	return nil
}

func (s *Service) List(ctx context.Context, ownerID pgtype.UUID) (ListResult, error) {
	if !ownerID.Valid {
		return ListResult{}, ErrForbidden
	}

	org, err := s.orgSvc.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, orgsvc.ErrOrgNotFound) {
			return ListResult{}, ErrOrgRequired
		}
		return ListResult{}, err
	}

	loc, err := s.orgLocation(ctx, org.ID)
	if err != nil {
		return ListResult{}, err
	}

	now := s.slotSvc.Now()
	todayStart := startOfDay(now, loc)
	tomorrowStart := todayStart.Add(24 * time.Hour)

	rows, err := s.q.ListScheduledBookingsByOrg(ctx, db.ListScheduledBookingsByOrgParams{
		OrganizationID: org.ID,
		StartAt:        timestamptz(todayStart),
	})
	if err != nil {
		return ListResult{}, err
	}

	today := make([]BookingView, 0)
	upcoming := make([]BookingView, 0)
	for _, row := range rows {
		view := toBookingView(row)
		if row.StartAt.Time.Before(tomorrowStart) {
			today = append(today, view)
		} else {
			upcoming = append(upcoming, view)
		}
	}

	return ListResult{Today: today, Upcoming: upcoming}, nil
}

func (s *Service) Cancel(ctx context.Context, ownerID, bookingID pgtype.UUID) error {
	if !ownerID.Valid {
		return ErrForbidden
	}
	if !bookingID.Valid {
		return ErrBookingNotFound
	}

	org, err := s.orgSvc.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, orgsvc.ErrOrgNotFound) {
			return ErrOrgRequired
		}
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	booking, err := qtx.GetScheduledBookingForUpdate(ctx, db.GetScheduledBookingForUpdateParams{
		ID:             bookingID,
		OrganizationID: org.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBookingNotFound
		}
		return err
	}

	if _, err := s.cancelScheduledBookingTx(ctx, qtx, org, booking, cancelledByRecruiter); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	s.deleteCalendarEventIfPresent(ctx, org, booking)

	return nil
}

func (s *Service) slotWindow(ctx context.Context, orgID pgtype.UUID) (time.Time, time.Time, error) {
	loc, err := s.orgLocation(ctx, orgID)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	now := s.slotSvc.Now()
	todayStart := startOfDay(now, loc)
	start := maxTime(now.UTC(), todayStart)
	end := todayStart.AddDate(0, 0, slotsvc.DefaultWindowDays)
	return start, end, nil
}

func (s *Service) orgLocation(ctx context.Context, orgID pgtype.UUID) (*time.Location, error) {
	loc := defaultLocation()
	settings, err := s.q.GetAvailabilitySettingsByOrg(ctx, orgID)
	if err == nil && settings.Timezone != "" {
		if parsed, err := time.LoadLocation(settings.Timezone); err == nil {
			loc = parsed
		}
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return loc, nil
}

func MeetingLocationFromType(it db.InterviewType) *string {
	if it.MeetingProvider != db.MeetingProviderOnSite || !it.MeetingUrl.Valid {
		return nil
	}
	loc := it.MeetingUrl.String
	return &loc
}

func bookingPayload(org db.Organization, it db.InterviewType, booking db.Booking, slot db.Slot, meetLink string) ([]byte, error) {
	data := map[string]any{
		"booking_id":           booking.ID.String(),
		"organization_name":    org.Name,
		"organization_slug":    org.Slug,
		"interview_type_title": it.Title,
		"interview_type_slug":  it.Slug,
		"candidate_name":       booking.CandidateName,
		"candidate_phone":      booking.CandidatePhone,
		"candidate_email":      booking.CandidateEmail,
		"slot_start_at":        slot.StartAt.Time.UTC().Format(time.RFC3339),
		"slot_end_at":          slot.EndAt.Time.UTC().Format(time.RFC3339),
		"reschedule_token":     booking.RescheduleToken,
		"cancel_token":         booking.CancelToken,
	}
	if meetLink != "" {
		data["meet_link"] = meetLink
	} else if booking.MeetLink.Valid {
		data["meet_link"] = booking.MeetLink.String
	}
	if it.MeetingProvider == db.MeetingProviderOnSite && it.MeetingUrl.Valid {
		data["meeting_location"] = it.MeetingUrl.String
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

func toBookingView(row db.ListScheduledBookingsByOrgRow) BookingView {
	return BookingView{
		ID:             row.ID.String(),
		SlotID:         row.SlotID.String(),
		Name:           row.CandidateName,
		Phone:          row.CandidatePhone,
		Email:          row.CandidateEmail,
		Status:         string(row.Status),
		StartAt:        row.StartAt.Time.UTC(),
		EndAt:          row.EndAt.Time.UTC(),
		InterviewTitle: row.InterviewTypeTitle,
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
