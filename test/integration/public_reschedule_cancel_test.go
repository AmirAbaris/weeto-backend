package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestRescheduleSuccess(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	slotA := env.FirstAvailableSlot(it)
	slots := env.ListSlots(it.ID)
	if len(slots) < 2 {
		t.Fatal("need at least 2 slots")
	}
	slotB := slots[1]

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slotA.ID,
		Name:   "Ali Rezaei",
		Phone:  "+989121234567",
		Email:  "ali@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	ctx, err := env.BookingSvc.GetRescheduleContext(env.Ctx, result.Booking.RescheduleToken)
	if err != nil {
		t.Fatalf("get reschedule context: %v", err)
	}
	if ctx.Booking.Name != "Ali Rezaei" {
		t.Fatalf("name = %q", ctx.Booking.Name)
	}
	if ctx.Booking.Phone != "+989121234567" {
		t.Fatalf("phone = %q", ctx.Booking.Phone)
	}
	if ctx.Booking.Email != "ali@example.com" {
		t.Fatalf("email = %q", ctx.Booking.Email)
	}
	if !ctx.CanModify {
		t.Fatal("expected can_modify true")
	}
	if len(ctx.Slots) == 0 {
		t.Fatal("expected available slots")
	}

	rescheduled, _, _, _, err := env.BookingSvc.Reschedule(env.Ctx, result.Booking.RescheduleToken, slotB.ID)
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if rescheduled.Booking.SlotID != slotB.ID {
		t.Fatalf("slot_id = %s, want %s", rescheduled.Booking.SlotID, slotB.ID)
	}

	freedA, err := env.Queries.GetSlotByID(env.Ctx, slotA.ID)
	if err != nil {
		t.Fatalf("get slot A: %v", err)
	}
	if freedA.Booked {
		t.Fatal("expected slot A to be free")
	}

	bookedB, err := env.Queries.GetSlotByID(env.Ctx, slotB.ID)
	if err != nil {
		t.Fatalf("get slot B: %v", err)
	}
	if !bookedB.Booked {
		t.Fatal("expected slot B to be booked")
	}

	outbox, err := env.Queries.ListNotificationOutboxByOrg(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	var rescheduledCount int
	for _, row := range outbox {
		if row.EventType == db.NotificationEventTypeBookingRescheduled {
			rescheduledCount++
		}
	}
	if rescheduledCount != 1 {
		t.Fatalf("booking_rescheduled outbox rows = %d, want 1", rescheduledCount)
	}

	for _, row := range outbox {
		if row.EventType != db.NotificationEventTypeBookingRescheduled {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload["recipient"] != "candidate" {
			t.Fatalf("recipient = %v, want candidate", payload["recipient"])
		}
	}
}

func TestRescheduleCutoffBlocked(t *testing.T) {
	loc := tehran(t)
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, loc)
	env := NewTestEnv(t, now)
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	slot := slotAtLocalHour(t, env, it, loc, 13)
	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Cutoff Test",
		Phone:  "+989121234567",
		Email:  "cutoff@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	ctx, err := env.BookingSvc.GetRescheduleContext(env.Ctx, result.Booking.RescheduleToken)
	if err != nil {
		t.Fatalf("get context: %v", err)
	}
	if ctx.CanModify {
		t.Fatal("expected can_modify false within cutoff")
	}

	target := slotAtLocalHour(t, env, it, loc, 15)
	_, _, _, _, err = env.BookingSvc.Reschedule(env.Ctx, result.Booking.RescheduleToken, target.ID)
	if err == nil {
		t.Fatal("expected cutoff error")
	}
	if err != bookingsvc.ErrModifyCutoff {
		t.Fatalf("err = %v, want ErrModifyCutoff", err)
	}
}

func TestRescheduleInvalidToken(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))

	_, err := env.BookingSvc.GetRescheduleContext(env.Ctx, "invalid-token")
	if err != bookingsvc.ErrTokenNotFound {
		t.Fatalf("get err = %v", err)
	}

	_, _, _, _, err = env.BookingSvc.Reschedule(env.Ctx, "invalid-token", pgtype.UUID{Valid: true, Bytes: [16]byte{1}})
	if err != bookingsvc.ErrTokenNotFound {
		t.Fatalf("post err = %v", err)
	}
}

func TestRescheduleSlotConflict(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	slots := env.ListSlots(it.ID)
	if len(slots) < 3 {
		t.Fatal("need at least 3 slots")
	}

	result1, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slots[0].ID,
		Name:   "First",
		Phone:  "+989121111111",
		Email:  "first@example.com",
	})
	if err != nil {
		t.Fatalf("book 1: %v", err)
	}

	result2, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slots[1].ID,
		Name:   "Second",
		Phone:  "+989122222222",
		Email:  "second@example.com",
	})
	if err != nil {
		t.Fatalf("book 2: %v", err)
	}

	target := slots[2].ID
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup

	for _, token := range []string{result1.Booking.RescheduleToken, result2.Booking.RescheduleToken} {
		wg.Add(1)
		go func(tok string) {
			defer wg.Done()
			<-start
			_, _, _, _, err := env.BookingSvc.Reschedule(env.Ctx, tok, target)
			errs <- err
		}(token)
	}

	close(start)
	wg.Wait()
	close(errs)

	var success, conflict int
	for err := range errs {
		switch err {
		case nil:
			success++
		case bookingsvc.ErrSlotUnavailable:
			conflict++
		default:
			t.Fatalf("unexpected err: %v", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d, want 1 each", success, conflict)
	}
}

func TestCancelByTokenSuccess(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Cancel Me",
		Phone:  "+989121234567",
		Email:  "cancel@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	cancelCtx, err := env.BookingSvc.GetCancelContext(env.Ctx, result.Booking.CancelToken)
	if err != nil {
		t.Fatalf("get cancel context: %v", err)
	}
	if cancelCtx.Booking.Name != "Cancel Me" {
		t.Fatalf("name = %q", cancelCtx.Booking.Name)
	}
	if !cancelCtx.CanModify {
		t.Fatal("expected can_modify true")
	}

	if err := env.BookingSvc.CancelByToken(env.Ctx, result.Booking.CancelToken); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	stored, err := env.Queries.GetBookingByID(env.Ctx, result.Booking.ID)
	if err != nil {
		t.Fatalf("get booking: %v", err)
	}
	if stored.Status != db.BookingStatusCancelled {
		t.Fatalf("status = %q, want cancelled", stored.Status)
	}

	freed, err := env.Queries.GetSlotByID(env.Ctx, slot.ID)
	if err != nil {
		t.Fatalf("get slot: %v", err)
	}
	if freed.Booked {
		t.Fatal("expected slot to be free")
	}

	outbox, err := env.Queries.ListNotificationOutboxByOrg(env.Ctx, env.OrgID)
	if err != nil {
		t.Fatalf("list outbox: %v", err)
	}
	var cancelledCount int
	for _, row := range outbox {
		if row.EventType == db.NotificationEventTypeBookingCancelled {
			cancelledCount++
		}
	}
	if cancelledCount != 2 {
		t.Fatalf("booking_cancelled outbox rows = %d, want 2", cancelledCount)
	}
}

func TestRebookAfterCancelByToken(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	first, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "First Booking",
		Phone:  "+989121234567",
		Email:  "first@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	if err := env.BookingSvc.CancelByToken(env.Ctx, first.Booking.CancelToken); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	second, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Second Booking",
		Phone:  "+989129876543",
		Email:  "second@example.com",
	})
	if err != nil {
		t.Fatalf("rebook: %v", err)
	}
	if second.Booking.ID == first.Booking.ID {
		t.Fatal("expected a new booking row after rebook")
	}
	if second.Booking.Status != db.BookingStatusScheduled {
		t.Fatalf("status = %q, want scheduled", second.Booking.Status)
	}

	stored, err := env.Queries.GetSlotByID(env.Ctx, slot.ID)
	if err != nil {
		t.Fatalf("get slot: %v", err)
	}
	if !stored.Booked {
		t.Fatal("expected slot to be booked again")
	}
}

func TestCancelByTokenCutoffBlocked(t *testing.T) {
	loc := tehran(t)
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, loc)
	env := NewTestEnv(t, now)
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	slot := slotAtLocalHour(t, env, it, loc, 13)
	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Cutoff Cancel",
		Phone:  "+989121234567",
		Email:  "cutoff-cancel@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	ctx, err := env.BookingSvc.GetCancelContext(env.Ctx, result.Booking.CancelToken)
	if err != nil {
		t.Fatalf("get context: %v", err)
	}
	if ctx.CanModify {
		t.Fatal("expected can_modify false")
	}

	err = env.BookingSvc.CancelByToken(env.Ctx, result.Booking.CancelToken)
	if err != bookingsvc.ErrModifyCutoff {
		t.Fatalf("err = %v, want ErrModifyCutoff", err)
	}
}

func TestCancelByTokenAlreadyUsed(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "Once",
		Phone:  "+989121234567",
		Email:  "once@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	if err := env.BookingSvc.CancelByToken(env.Ctx, result.Booking.CancelToken); err != nil {
		t.Fatalf("first cancel: %v", err)
	}

	err = env.BookingSvc.CancelByToken(env.Ctx, result.Booking.CancelToken)
	if err != bookingsvc.ErrTokenNotFound {
		t.Fatalf("second cancel err = %v, want ErrTokenNotFound", err)
	}
}

func TestRescheduleHTTP(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	slotA := env.FirstAvailableSlot(it)
	slots := env.ListSlots(it.ID)
	slotB := slots[1]

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slotA.ID,
		Name:   "HTTP Reschedule",
		Phone:  "+989121234567",
		Email:  "http-reschedule@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	ts := newTestServer(t, env)
	defer ts.Close()

	getResp, err := http.Get(ts.URL + "/public/reschedule/" + result.Booking.RescheduleToken)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getResp.Body)
		t.Fatalf("GET status = %d, body = %s", getResp.StatusCode, body)
	}

	body, err := json.Marshal(map[string]string{"slot_id": slotB.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	postResp, err := http.Post(ts.URL+"/public/reschedule/"+result.Booking.RescheduleToken, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(postResp.Body)
		t.Fatalf("POST status = %d, body = %s", postResp.StatusCode, respBody)
	}
}

func TestCancelHTTP(t *testing.T) {
	env := NewTestEnv(t, mondayMorningTehran(t))
	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))
	slot := env.FirstAvailableSlot(it)

	result, err := env.BookingSvc.Book(env.Ctx, env.OrgSlug, it.Slug, bookingsvc.BookInput{
		SlotID: slot.ID,
		Name:   "HTTP Cancel",
		Phone:  "+989121234567",
		Email:  "http-cancel@example.com",
	})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	ts := newTestServer(t, env)
	defer ts.Close()

	getResp, err := http.Get(ts.URL + "/public/cancel/" + result.Booking.CancelToken)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getResp.Body)
		t.Fatalf("GET status = %d, body = %s", getResp.StatusCode, body)
	}

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/public/cancel/"+result.Booking.CancelToken, nil)
	if err != nil {
		t.Fatal(err)
	}
	postResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(postResp.Body)
		t.Fatalf("POST status = %d, body = %s", postResp.StatusCode, body)
	}
}

func slotAtLocalHour(t *testing.T, env *TestEnv, it db.InterviewType, loc *time.Location, hour int) db.Slot {
	t.Helper()
	for _, slot := range env.ListSlots(it.ID) {
		if slot.StartAt.Time.In(loc).Hour() == hour {
			return slot
		}
	}
	t.Fatalf("no slot at hour %d", hour)
	return db.Slot{}
}
