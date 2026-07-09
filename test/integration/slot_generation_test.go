package integration

import (
	"testing"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/test/fixtures"
)

func tehran(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func mondayMorningTehran(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2026, 7, 6, 9, 0, 0, 0, tehran(t))
}

func TestSlotGenerationBasicGrid(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	slots := env.SlotsOnLocalDay(it.ID, day, loc)
	if len(slots) != 8 {
		t.Fatalf("got %d slots, want 8", len(slots))
	}
	for i, slot := range slots {
		local := slot.StartAt.Time.In(loc)
		wantHour := 9 + i
		if local.Hour() != wantHour || local.Minute() != 0 {
			t.Fatalf("slot %d at %v, want %02d:00", i, local.Format("15:04"), wantHour)
		}
	}
}

func TestSlotGenerationBufferSpacing(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(30, 15)
	in := fixtures.Monday9to17(50)
	in.WorkingHours[0].EndTime = "11:00"
	env.UpsertAvailability(in)

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	slots := env.SlotsOnLocalDay(it.ID, day, loc)
	if len(slots) != 3 {
		t.Fatalf("got %d slots, want 3", len(slots))
	}
	gap := slots[1].StartAt.Time.Sub(slots[0].StartAt.Time)
	if gap != 45*time.Minute {
		t.Fatalf("gap = %v, want 45m", gap)
	}
}

func TestSlotGenerationLunchBreak(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17WithLunch(50))

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	slots := env.SlotsOnLocalDay(it.ID, day, loc)
	for _, slot := range slots {
		local := slot.StartAt.Time.In(loc)
		if local.Hour() == 12 {
			t.Fatalf("slot overlaps lunch at %v", local.Format("15:04"))
		}
	}
}

func TestSlotGenerationMaxPerDay(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(30, 0)
	env.UpsertAvailability(fixtures.Monday9to17(2))

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	slots := env.SlotsOnLocalDay(it.ID, day, loc)
	if len(slots) != 2 {
		t.Fatalf("got %d slots, want 2", len(slots))
	}
}

func TestSlotGenerationTimeOffBlock(t *testing.T) {
	loc := tehran(t)
	now := time.Date(2026, 7, 7, 8, 0, 0, 0, loc) // Tuesday
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	in := fixtures.WithTimeOff(
		fixtures.Tuesday9to17(50),
		"2026-07-07T10:00:00Z",
		"2026-07-07T14:00:00Z",
	)
	env.UpsertAvailability(in)

	day := time.Date(2026, 7, 7, 0, 0, 0, 0, loc)
	slots := env.SlotsOnLocalDay(it.ID, day, loc)
	blockStart := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	blockEnd := time.Date(2026, 7, 7, 14, 0, 0, 0, time.UTC)
	for _, slot := range slots {
		start := slot.StartAt.Time.UTC()
		end := slot.EndAt.Time.UTC()
		if start.Before(blockEnd) && end.After(blockStart) {
			t.Fatalf("slot overlaps time off: %v-%v", start, end)
		}
	}
}

func TestSlotGenerationRollingWindow(t *testing.T) {
	loc := tehran(t)
	now := time.Date(2026, 7, 6, 15, 0, 0, 0, loc)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	slots := env.SlotsOnLocalDay(it.ID, day, loc)
	for _, slot := range slots {
		local := slot.StartAt.Time.In(loc)
		if local.Hour() < 15 {
			t.Fatalf("past slot should be excluded: %v", local.Format(time.RFC3339))
		}
	}

	all := env.ListSlots(it.ID)
	windowEnd := time.Date(2026, 7, 20, 0, 0, 0, 0, loc)
	for _, slot := range all {
		if !slot.StartAt.Time.Before(windowEnd) {
			t.Fatalf("slot beyond 14-day window: %v", slot.StartAt.Time)
		}
	}
}

func TestSlotGenerationCustomHorizon(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.WithHorizon(fixtures.Monday9to17(50), 3))

	all := env.ListSlots(it.ID)
	windowEnd := time.Date(2026, 7, 9, 0, 0, 0, 0, loc)
	for _, slot := range all {
		if !slot.StartAt.Time.Before(windowEnd) {
			t.Fatalf("slot beyond 3-day window: %v", slot.StartAt.Time)
		}
	}
}

func TestSlotGenerationHorizonShrink(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	before := env.ListSlots(it.ID)
	if len(before) == 0 {
		t.Fatal("expected slots before horizon shrink")
	}

	env.UpsertAvailability(fixtures.WithHorizon(fixtures.Monday9to17(50), 3))

	after := env.ListSlots(it.ID)
	windowEnd := time.Date(2026, 7, 9, 0, 0, 0, 0, loc)
	for _, slot := range after {
		if !slot.StartAt.Time.Before(windowEnd) {
			t.Fatalf("slot beyond shrunk horizon: %v", slot.StartAt.Time)
		}
	}
}

func TestSlotGenerationBookedSlotPreserved(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(50))

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	before := env.SlotsOnLocalDay(it.ID, day, loc)
	if len(before) == 0 {
		t.Fatal("expected slots before booking")
	}
	booked := before[0]
	env.BookSlot(booked.ID)

	env.UpsertAvailability(fixtures.Monday9to17(50))

	after := env.SlotsOnLocalDay(it.ID, day, loc)
	var found bool
	for _, slot := range after {
		if slot.ID.Bytes == booked.ID.Bytes && slot.Booked {
			found = true
		}
	}
	if !found {
		t.Fatal("booked slot was not preserved")
	}
}

func TestSlotGenerationSecondInterviewType(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	first := env.CreateInterviewType(60, 0)
	env.UpsertAvailability(fixtures.Monday9to17(8))

	second := env.CreateInterviewType(60, 0)

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	firstSlots := env.SlotsOnLocalDay(first.ID, day, loc)
	if len(firstSlots) != 8 {
		t.Fatalf("first type: got %d slots, want 8", len(firstSlots))
	}

	secondSlots := env.SlotsOnLocalDay(second.ID, day, loc)
	if len(secondSlots) != 8 {
		t.Fatalf("second type: got %d slots, want 8", len(secondSlots))
	}
}

func TestSlotGenerationInterviewTypeBufferChange(t *testing.T) {
	loc := tehran(t)
	now := mondayMorningTehran(t)
	env := NewTestEnv(t, now)

	it := env.CreateInterviewType(30, 0)
	in := fixtures.Monday9to17(50)
	in.WorkingHours[0].EndTime = "11:00"
	env.UpsertAvailability(in)

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, loc)
	before := env.SlotsOnLocalDay(it.ID, day, loc)
	if len(before) < 2 {
		t.Fatalf("need at least 2 slots, got %d", len(before))
	}
	gapBefore := before[1].StartAt.Time.Sub(before[0].StartAt.Time)
	if gapBefore != 30*time.Minute {
		t.Fatalf("initial gap = %v, want 30m", gapBefore)
	}

	_, err := env.Queries.UpdateInterviewType(env.Ctx, db.UpdateInterviewTypeParams{
		ID:              it.ID,
		Title:           it.Title,
		Slug:            it.Slug,
		DurationMinutes: 30,
		BufferMinutes:   30,
		MeetingProvider: it.MeetingProvider,
		MeetingUrl:      it.MeetingUrl,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := env.SlotSvc.RegenerateForType(env.Ctx, nil, env.OrgID, it.ID, 30, 30); err != nil {
		t.Fatal(err)
	}

	after := env.SlotsOnLocalDay(it.ID, day, loc)
	if len(after) < 2 {
		t.Fatalf("need at least 2 slots after buffer change, got %d", len(after))
	}
	gapAfter := after[1].StartAt.Time.Sub(after[0].StartAt.Time)
	if gapAfter != 60*time.Minute {
		t.Fatalf("gap after buffer change = %v, want 60m", gapAfter)
	}
}
