package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
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
