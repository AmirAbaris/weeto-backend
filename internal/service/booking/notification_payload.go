package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type cancelledBy string

const (
	cancelledByCandidate  cancelledBy = "candidate"
	cancelledByRecruiter  cancelledBy = "recruiter"
)

func notificationPayload(
	ctx context.Context,
	q *db.Queries,
	org db.Organization,
	it db.InterviewType,
	booking db.Booking,
	slot db.Slot,
	meetLink, recipient string,
	extra map[string]any,
) ([]byte, error) {
	base, err := bookingPayload(org, it, booking, slot, meetLink)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(base, &data); err != nil {
		return nil, err
	}

	owner, err := q.GetUserByID(ctx, org.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("recruiter email: %w", err)
	}

	data["recipient"] = recipient
	data["recruiter_email"] = owner.Email
	for k, v := range extra {
		data[k] = v
	}
	return json.Marshal(data)
}

func rescheduleNotificationPayload(
	ctx context.Context,
	q *db.Queries,
	org db.Organization,
	it db.InterviewType,
	booking db.Booking,
	slot, prevSlot db.Slot,
) ([]byte, error) {
	return notificationPayload(ctx, q, org, it, booking, slot, "", "candidate", map[string]any{
		"previous_slot_start_at": prevSlot.StartAt.Time.UTC().Format(time.RFC3339),
		"previous_slot_end_at":   prevSlot.EndAt.Time.UTC().Format(time.RFC3339),
	})
}

func cancelNotificationPayloads(
	ctx context.Context,
	q *db.Queries,
	org db.Organization,
	it db.InterviewType,
	booking db.Booking,
	slot db.Slot,
	by cancelledBy,
) (candidate []byte, recruiter []byte, err error) {
	extra := map[string]any{"cancelled_by": string(by)}
	candidate, err = notificationPayload(ctx, q, org, it, booking, slot, "", "candidate", extra)
	if err != nil {
		return nil, nil, err
	}
	recruiter, err = notificationPayload(ctx, q, org, it, booking, slot, "", "recruiter", extra)
	if err != nil {
		return nil, nil, err
	}
	return candidate, recruiter, nil
}

func insertCancelNotifications(
	ctx context.Context,
	qtx *db.Queries,
	org db.Organization,
	it db.InterviewType,
	booking db.Booking,
	slot db.Slot,
	by cancelledBy,
) error {
	if err := cancelPendingReminders(ctx, qtx, booking.ID); err != nil {
		return err
	}

	candidatePayload, recruiterPayload, err := cancelNotificationPayloads(ctx, qtx, org, it, booking, slot, by)
	if err != nil {
		return err
	}
	for _, payload := range [][]byte{candidatePayload, recruiterPayload} {
		if _, err := qtx.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
			OrganizationID: org.ID,
			EventType:      db.NotificationEventTypeBookingCancelled,
			Payload:        payload,
		}); err != nil {
			return err
		}
	}
	return nil
}

func reminderScheduledAt(slotStart, now time.Time) (time.Time, bool) {
	at := slotStart.Add(-24 * time.Hour)
	if !at.After(now) {
		return time.Time{}, false
	}
	return at, true
}

func insertReminderNotification(
	ctx context.Context,
	qtx *db.Queries,
	org db.Organization,
	it db.InterviewType,
	booking db.Booking,
	slot db.Slot,
	meetLink string,
	now time.Time,
) error {
	scheduledAt, ok := reminderScheduledAt(slot.StartAt.Time, now)
	if !ok {
		return nil
	}

	payload, err := notificationPayload(ctx, qtx, org, it, booking, slot, meetLink, "candidate", nil)
	if err != nil {
		return err
	}

	_, err = qtx.InsertNotificationOutbox(ctx, db.InsertNotificationOutboxParams{
		OrganizationID: org.ID,
		EventType:      db.NotificationEventTypeReminder24h,
		Payload:        payload,
		ScheduledAt:    pgtype.Timestamptz{Time: scheduledAt.UTC(), Valid: true},
	})
	return err
}

func cancelPendingReminders(ctx context.Context, qtx *db.Queries, bookingID pgtype.UUID) error {
	_, err := qtx.CancelPendingRemindersByBookingID(ctx, bookingID.String())
	return err
}
