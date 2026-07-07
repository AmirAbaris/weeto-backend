package booking

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	googleplatform "github.com/AmirAbaris/weeto-backend/internal/platform/google"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ModifyBookingView struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Phone   string    `json:"phone"`
	Email   string    `json:"email"`
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
	Status  string    `json:"status"`
}

type ModifyOrganizationView struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ModifyInterviewTypeView struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	DurationMinutes int32  `json:"duration_minutes"`
}

type RescheduleContext struct {
	Booking       ModifyBookingView       `json:"booking"`
	Organization  ModifyOrganizationView  `json:"organization"`
	InterviewType ModifyInterviewTypeView `json:"interview_type"`
	CurrentSlot   SlotView                `json:"current_slot"`
	Slots         []SlotView              `json:"slots"`
	CanModify     bool                    `json:"can_modify"`
	CutoffHours   int                     `json:"cutoff_hours"`
}

type CancelContext struct {
	Booking       ModifyBookingView       `json:"booking"`
	Organization  ModifyOrganizationView  `json:"organization"`
	InterviewType ModifyInterviewTypeView `json:"interview_type"`
	CanModify     bool                    `json:"can_modify"`
	CutoffHours   int                     `json:"cutoff_hours"`
}

func (s *Service) GetRescheduleContext(ctx context.Context, token string) (RescheduleContext, error) {
	if token == "" {
		return RescheduleContext{}, ErrTokenNotFound
	}

	row, err := s.q.GetScheduledBookingByRescheduleToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RescheduleContext{}, ErrTokenNotFound
		}
		return RescheduleContext{}, err
	}

	org, it, err := s.loadBookingMeta(ctx, row.OrganizationID, row.InterviewTypeID)
	if err != nil {
		return RescheduleContext{}, err
	}

	slots, err := s.listAvailableSlotsForType(ctx, row.OrganizationID, row.InterviewTypeID)
	if err != nil {
		return RescheduleContext{}, err
	}

	canModify := s.canModifyBooking(row.SlotStartAt.Time)

	return RescheduleContext{
		Booking:       toModifyBookingView(row.ID, row.CandidateName, row.CandidatePhone, row.CandidateEmail, row.Status, row.SlotStartAt, row.SlotEndAt),
		Organization:  toModifyOrgView(org),
		InterviewType: toModifyTypeView(it),
		CurrentSlot: SlotView{
			ID:      row.SlotID.String(),
			StartAt: row.SlotStartAt.Time.UTC(),
			EndAt:   row.SlotEndAt.Time.UTC(),
		},
		Slots:       slots,
		CanModify:   canModify,
		CutoffHours: DefaultModifyCutoffHours,
	}, nil
}

func (s *Service) Reschedule(ctx context.Context, token string, newSlotID pgtype.UUID) (BookingResult, string, string, *string, error) {
	if token == "" {
		return BookingResult{}, "", "", nil, ErrTokenNotFound
	}
	if !newSlotID.Valid {
		return BookingResult{}, "", "", nil, ErrInvalidSlotID
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return BookingResult{}, "", "", nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	row, err := qtx.GetScheduledBookingByRescheduleTokenForUpdate(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BookingResult{}, "", "", nil, ErrTokenNotFound
		}
		return BookingResult{}, "", "", nil, err
	}

	if !s.canModifyBooking(row.SlotStartAt.Time) {
		return BookingResult{}, "", "", nil, ErrModifyCutoff
	}

	if row.SlotID == newSlotID {
		return BookingResult{}, "", "", nil, ErrSameSlot
	}

	oldSlot, err := qtx.GetSlotByID(ctx, row.SlotID)
	if err != nil {
		return BookingResult{}, "", "", nil, err
	}

	newSlot, err := qtx.GetSlotForUpdate(ctx, db.GetSlotForUpdateParams{
		ID:              newSlotID,
		OrganizationID:  row.OrganizationID,
		InterviewTypeID: row.InterviewTypeID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BookingResult{}, "", "", nil, ErrSlotUnavailable
		}
		return BookingResult{}, "", "", nil, err
	}

	if err := qtx.SetSlotBooked(ctx, db.SetSlotBookedParams{
		ID:     oldSlot.ID,
		Booked: false,
	}); err != nil {
		return BookingResult{}, "", "", nil, err
	}

	rows, err := qtx.MarkSlotBooked(ctx, newSlot.ID)
	if err != nil {
		return BookingResult{}, "", "", nil, err
	}
	if rows == 0 {
		return BookingResult{}, "", "", nil, ErrSlotUnavailable
	}

	updated, err := qtx.UpdateBookingSlot(ctx, db.UpdateBookingSlotParams{
		ID:     row.ID,
		SlotID: newSlot.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BookingResult{}, "", "", nil, ErrBookingNotModifiable
		}
		return BookingResult{}, "", "", nil, err
	}

	org, it, err := s.loadBookingMetaTx(ctx, qtx, row.OrganizationID, row.InterviewTypeID)
	if err != nil {
		return BookingResult{}, "", "", nil, err
	}

	payload, err := reschedulePayload(org, it, updated, newSlot, oldSlot)
	if err != nil {
		return BookingResult{}, "", "", nil, err
	}

	if _, err := qtx.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
		OrganizationID: org.ID,
		EventType:      db.NotificationEventTypeBookingRescheduled,
		Payload:        payload,
	}); err != nil {
		return BookingResult{}, "", "", nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return BookingResult{}, "", "", nil, err
	}

	if updated.CalendarEventID.Valid && updated.CalendarEventID.String != "" {
		if err := s.calendar.UpdateInterviewEvent(ctx, org.OwnerID, updated.CalendarEventID.String, googleplatform.EventInput{
			Summary:        it.Title + " — " + updated.CandidateName,
			CandidateEmail: updated.CandidateEmail,
			StartAt:        newSlot.StartAt.Time,
			EndAt:          newSlot.EndAt.Time,
		}); err != nil {
			slog.Error("update calendar event failed", "booking_id", updated.ID, "event_id", updated.CalendarEventID.String, "err", err)
		}
	}

	return BookingResult{Booking: updated, Slot: newSlot}, org.Name, it.Title, MeetingLocationFromType(it), nil
}

func (s *Service) GetCancelContext(ctx context.Context, token string) (CancelContext, error) {
	if token == "" {
		return CancelContext{}, ErrTokenNotFound
	}

	row, err := s.q.GetScheduledBookingByCancelToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CancelContext{}, ErrTokenNotFound
		}
		return CancelContext{}, err
	}

	org, it, err := s.loadBookingMeta(ctx, row.OrganizationID, row.InterviewTypeID)
	if err != nil {
		return CancelContext{}, err
	}

	return CancelContext{
		Booking:       toModifyBookingView(row.ID, row.CandidateName, row.CandidatePhone, row.CandidateEmail, row.Status, row.SlotStartAt, row.SlotEndAt),
		Organization:  toModifyOrgView(org),
		InterviewType: toModifyTypeView(it),
		CanModify:     s.canModifyBooking(row.SlotStartAt.Time),
		CutoffHours:   DefaultModifyCutoffHours,
	}, nil
}

func (s *Service) CancelByToken(ctx context.Context, token string) error {
	if token == "" {
		return ErrTokenNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	row, err := qtx.GetScheduledBookingByCancelTokenForUpdate(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTokenNotFound
		}
		return err
	}

	if !s.canModifyBooking(row.SlotStartAt.Time) {
		return ErrModifyCutoff
	}

	org, err := s.q.GetOrganizationByID(ctx, row.OrganizationID)
	if err != nil {
		return err
	}

	booking := cancelRowToBooking(row)
	result, err := s.cancelScheduledBookingTx(ctx, qtx, org, booking)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	s.deleteCalendarEventIfPresent(ctx, org, booking)

	_ = result
	return nil
}

type cancelTxResult struct {
	cancelled db.Booking
	slot      db.Slot
}

func (s *Service) cancelScheduledBookingTx(ctx context.Context, qtx *db.Queries, org db.Organization, booking db.Booking) (cancelTxResult, error) {
	cancelled, err := qtx.CancelBooking(ctx, db.CancelBookingParams{
		ID:             booking.ID,
		OrganizationID: org.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cancelTxResult{}, ErrBookingNotFound
		}
		return cancelTxResult{}, err
	}

	if err := qtx.SetSlotBooked(ctx, db.SetSlotBookedParams{
		ID:     booking.SlotID,
		Booked: false,
	}); err != nil {
		return cancelTxResult{}, err
	}

	it, err := qtx.GetInterviewTypeByID(ctx, booking.InterviewTypeID)
	if err != nil {
		return cancelTxResult{}, err
	}

	slot, err := qtx.GetSlotByID(ctx, booking.SlotID)
	if err != nil {
		return cancelTxResult{}, err
	}

	payload, err := bookingPayload(org, it, cancelled, slot, "", "candidate")
	if err != nil {
		return cancelTxResult{}, err
	}

	if _, err := qtx.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
		OrganizationID: org.ID,
		EventType:      db.NotificationEventTypeBookingCancelled,
		Payload:        payload,
	}); err != nil {
		return cancelTxResult{}, err
	}

	return cancelTxResult{cancelled: cancelled, slot: slot}, nil
}

func (s *Service) deleteCalendarEventIfPresent(ctx context.Context, org db.Organization, booking db.Booking) {
	if booking.CalendarEventID.Valid && booking.CalendarEventID.String != "" {
		if err := s.calendar.DeleteEvent(ctx, org.OwnerID, booking.CalendarEventID.String); err != nil {
			slog.Error("delete calendar event failed", "booking_id", booking.ID, "event_id", booking.CalendarEventID.String, "err", err)
		}
	}
}

func (s *Service) listAvailableSlotsForType(ctx context.Context, orgID, typeID pgtype.UUID) ([]SlotView, error) {
	windowStart, windowEnd, err := s.slotWindow(ctx, orgID)
	if err != nil {
		return nil, err
	}

	slots, err := s.q.ListAvailableSlotsByType(ctx, db.ListAvailableSlotsByTypeParams{
		OrganizationID:  orgID,
		InterviewTypeID: typeID,
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

func (s *Service) loadBookingMeta(ctx context.Context, orgID, typeID pgtype.UUID) (db.Organization, db.InterviewType, error) {
	org, err := s.q.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return db.Organization{}, db.InterviewType{}, err
	}
	it, err := s.q.GetInterviewTypeByID(ctx, typeID)
	if err != nil {
		return db.Organization{}, db.InterviewType{}, err
	}
	return org, it, nil
}

func (s *Service) loadBookingMetaTx(ctx context.Context, qtx *db.Queries, orgID, typeID pgtype.UUID) (db.Organization, db.InterviewType, error) {
	org, err := qtx.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return db.Organization{}, db.InterviewType{}, err
	}
	it, err := qtx.GetInterviewTypeByID(ctx, typeID)
	if err != nil {
		return db.Organization{}, db.InterviewType{}, err
	}
	return org, it, nil
}

func toModifyBookingView(id pgtype.UUID, name, phone, email string, status db.BookingStatus, startAt, endAt pgtype.Timestamptz) ModifyBookingView {
	return ModifyBookingView{
		ID:      id.String(),
		Name:    name,
		Phone:   phone,
		Email:   email,
		StartAt: startAt.Time.UTC(),
		EndAt:   endAt.Time.UTC(),
		Status:  string(status),
	}
}

func toModifyOrgView(org db.Organization) ModifyOrganizationView {
	return ModifyOrganizationView{Name: org.Name, Slug: org.Slug}
}

func toModifyTypeView(it db.InterviewType) ModifyInterviewTypeView {
	return ModifyInterviewTypeView{
		Title:           it.Title,
		Slug:            it.Slug,
		DurationMinutes: it.DurationMinutes,
	}
}

func cancelRowToBooking(row db.GetScheduledBookingByCancelTokenForUpdateRow) db.Booking {
	return db.Booking{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		InterviewTypeID: row.InterviewTypeID,
		SlotID:          row.SlotID,
		CandidateName:   row.CandidateName,
		CandidatePhone:  row.CandidatePhone,
		CandidateEmail:  row.CandidateEmail,
		Status:          row.Status,
		MeetLink:        row.MeetLink,
		CalendarEventID: row.CalendarEventID,
		RescheduleToken: row.RescheduleToken,
		CancelToken:     row.CancelToken,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func reschedulePayload(org db.Organization, it db.InterviewType, booking db.Booking, slot, prevSlot db.Slot) ([]byte, error) {
	payload, err := bookingPayload(org, it, booking, slot, "", "candidate")
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	data["previous_slot_start_at"] = prevSlot.StartAt.Time.UTC().Format(time.RFC3339)
	data["previous_slot_end_at"] = prevSlot.EndAt.Time.UTC().Format(time.RFC3339)
	return json.Marshal(data)
}
