package slot

import (
	"testing"
	"time"
)

func tehranLoc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func clock(h, m int) time.Duration {
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
}

func mondayNineAMTehran(t *testing.T) time.Time {
	t.Helper()
	loc := tehranLoc(t)
	// 2026-07-06 is a Monday.
	return time.Date(2026, 7, 6, 9, 0, 0, 0, loc)
}

func TestSubtractIntervalsBreak(t *testing.T) {
	base := []localInterval{{start: clock(9, 0), end: clock(17, 0)}}
	cuts := []localInterval{{start: clock(12, 0), end: clock(13, 0)}}

	got := subtractIntervals(base, cuts)
	if len(got) != 2 {
		t.Fatalf("got %d intervals, want 2", len(got))
	}
	if got[0].start != clock(9, 0) || got[0].end != clock(12, 0) {
		t.Fatalf("morning interval = %+v", got[0])
	}
	if got[1].start != clock(13, 0) || got[1].end != clock(17, 0) {
		t.Fatalf("afternoon interval = %+v", got[1])
	}
}

func TestTimeOffIntervalsForDayPartialOverlap(t *testing.T) {
	loc := tehranLoc(t)
	day := time.Date(2026, 7, 7, 0, 0, 0, 0, loc) // Tuesday

	blocks := []TimeOff{
		{
			StartAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
			EndAt:   time.Date(2026, 7, 7, 14, 0, 0, 0, time.UTC),
		},
	}

	cuts := timeOffIntervalsForDay(day, loc, blocks)
	if len(cuts) != 1 {
		t.Fatalf("got %d cuts, want 1", len(cuts))
	}

	dayStart := startOfDay(day, loc)
	wantStart := time.Date(2026, 7, 7, 13, 30, 0, 0, loc).Sub(dayStart) // 10:00Z = 13:30 Tehran
	wantEnd := time.Date(2026, 7, 7, 17, 30, 0, 0, loc).Sub(dayStart)   // 14:00Z = 17:30 Tehran

	if cuts[0].start != wantStart || cuts[0].end != wantEnd {
		t.Fatalf("cut = %+v, want [%v, %v)", cuts[0], wantStart, wantEnd)
	}
}

func TestGenerateBasicGrid(t *testing.T) {
	loc := tehranLoc(t)
	now := mondayNineAMTehran(t)

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          14,
		Location:            loc,
		MaxInterviewsPerDay: 99,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 1, Start: clock(9, 0), End: clock(17, 0)},
		},
		DurationMinutes: 60,
		BufferMinutes:   0,
	})

	if len(candidates) < 8 {
		t.Fatalf("expected at least 8 slots on first Monday, got %d", len(candidates))
	}

	first := candidates[0].StartAt.In(loc)
	if first.Hour() != 9 || first.Minute() != 0 {
		t.Fatalf("first slot local time = %v, want 09:00", first.Format("15:04"))
	}
	if candidates[0].EndAt.Sub(candidates[0].StartAt) != time.Hour {
		t.Fatalf("slot duration = %v, want 1h", candidates[0].EndAt.Sub(candidates[0].StartAt))
	}

	// 9..16 local inclusive with 60m duration inside 9-17 window.
	var mondaySlots int
	for _, c := range candidates {
		local := c.StartAt.In(loc)
		if local.Year() == 2026 && local.Month() == time.July && local.Day() == 6 {
			mondaySlots++
			if local.Hour() < 9 || local.Hour() > 16 {
				t.Fatalf("slot outside 9-17 window: %v", local.Format(time.RFC3339))
			}
		}
	}
	if mondaySlots != 8 {
		t.Fatalf("monday slot count = %d, want 8", mondaySlots)
	}
}

func TestGenerateBufferSpacing(t *testing.T) {
	loc := tehranLoc(t)
	now := mondayNineAMTehran(t)

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          1,
		Location:            loc,
		MaxInterviewsPerDay: 99,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 1, Start: clock(9, 0), End: clock(11, 0)},
		},
		DurationMinutes: 30,
		BufferMinutes:   15,
	})

	if len(candidates) != 3 {
		t.Fatalf("got %d slots, want 3 (9:00, 9:45, 10:30 in a 2h window)", len(candidates))
	}
	gap := candidates[1].StartAt.Sub(candidates[0].StartAt)
	if gap != 45*time.Minute {
		t.Fatalf("gap = %v, want 45m", gap)
	}
}

func TestGenerateLunchBreak(t *testing.T) {
	loc := tehranLoc(t)
	now := mondayNineAMTehran(t)

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          1,
		Location:            loc,
		MaxInterviewsPerDay: 99,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 1, Start: clock(9, 0), End: clock(17, 0)},
		},
		Breaks: []Break{
			{DayOfWeek: 1, Start: clock(12, 0), End: clock(13, 0)},
		},
		DurationMinutes: 60,
		BufferMinutes:   0,
	})

	for _, c := range candidates {
		local := c.StartAt.In(loc)
		if local.Year() == 2026 && local.Month() == time.July && local.Day() == 6 {
			if local.Hour() == 12 {
				t.Fatalf("slot overlaps lunch break at %v", local.Format("15:04"))
			}
		}
	}
}

func TestGenerateMaxPerDay(t *testing.T) {
	loc := tehranLoc(t)
	now := mondayNineAMTehran(t)

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          1,
		Location:            loc,
		MaxInterviewsPerDay: 2,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 1, Start: clock(9, 0), End: clock(17, 0)},
		},
		DurationMinutes: 30,
		BufferMinutes:   0,
	})

	var mondaySlots int
	for _, c := range candidates {
		local := c.StartAt.In(loc)
		if local.Year() == 2026 && local.Month() == time.July && local.Day() == 6 {
			mondaySlots++
		}
	}
	if mondaySlots != 2 {
		t.Fatalf("monday slot count = %d, want 2", mondaySlots)
	}
}

func TestGenerateTimeOffBlock(t *testing.T) {
	loc := tehranLoc(t)
	now := time.Date(2026, 7, 7, 8, 0, 0, 0, loc) // Tuesday

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          1,
		Location:            loc,
		MaxInterviewsPerDay: 99,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 2, Start: clock(9, 0), End: clock(17, 0)},
		},
		TimeOff: []TimeOff{
			{
				StartAt: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
				EndAt:   time.Date(2026, 7, 7, 14, 0, 0, 0, time.UTC),
			},
		},
		DurationMinutes: 60,
		BufferMinutes:   0,
	})

	for _, c := range candidates {
		start := c.StartAt.UTC()
		end := c.EndAt.UTC()
		if start.Before(time.Date(2026, 7, 7, 14, 0, 0, 0, time.UTC)) &&
			end.After(time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)) {
			t.Fatalf("slot overlaps time off: %v - %v", start, end)
		}
	}
}

func TestGenerateRollingWindow(t *testing.T) {
	loc := tehranLoc(t)
	// Monday 2026-07-06 15:00 Tehran — slots before now on Monday should be excluded.
	now := time.Date(2026, 7, 6, 15, 0, 0, 0, loc)

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          14,
		Location:            loc,
		MaxInterviewsPerDay: 99,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 1, Start: clock(9, 0), End: clock(17, 0)},
		},
		DurationMinutes: 60,
		BufferMinutes:   0,
	})

	windowStart, windowEnd := regenWindow(now, loc, 14)
	for _, c := range candidates {
		if c.StartAt.Before(windowStart) || !c.StartAt.Before(windowEnd) {
			t.Fatalf("slot %v outside window [%v, %v)", c.StartAt, windowStart, windowEnd)
		}
	}

	for _, c := range candidates {
		local := c.StartAt.In(loc)
		if local.Year() == 2026 && local.Month() == time.July && local.Day() == 6 && local.Hour() < 15 {
			t.Fatalf("past slot on today should be excluded: %v", local.Format(time.RFC3339))
		}
	}

	// Window ends at start of 2026-07-20 Tehran midnight.
	if !windowEnd.Equal(time.Date(2026, 7, 20, 0, 0, 0, 0, loc)) {
		t.Fatalf("windowEnd = %v", windowEnd)
	}
}

func TestGenerateSkipsOccupiedStarts(t *testing.T) {
	loc := tehranLoc(t)
	now := mondayNineAMTehran(t)
	occupiedAt := time.Date(2026, 7, 6, 5, 30, 0, 0, time.UTC) // 09:00 Tehran

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          1,
		Location:            loc,
		MaxInterviewsPerDay: 99,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 1, Start: clock(9, 0), End: clock(12, 0)},
		},
		DurationMinutes: 60,
		BufferMinutes:   0,
		OccupiedStarts: map[time.Time]struct{}{
			occupiedAt: {},
		},
	})

	for _, c := range candidates {
		if c.StartAt.Equal(occupiedAt) {
			t.Fatal("generated slot at occupied start")
		}
	}
}

func TestGenerateRespectsExistingDayCount(t *testing.T) {
	loc := tehranLoc(t)
	now := mondayNineAMTehran(t)
	day := startOfDay(now, loc)

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          1,
		Location:            loc,
		MaxInterviewsPerDay: 3,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 1, Start: clock(9, 0), End: clock(17, 0)},
		},
		DurationMinutes: 30,
		BufferMinutes:   0,
		SlotsPerLocalDay: func(d time.Time) int32 {
			if d.Equal(day) {
				return 2
			}
			return 0
		},
	})

	var mondaySlots int
	for _, c := range candidates {
		if startOfDay(c.StartAt, loc).Equal(day) {
			mondaySlots++
		}
	}
	if mondaySlots != 1 {
		t.Fatalf("monday slot count = %d, want 1 (max 3 minus 2 existing)", mondaySlots)
	}
}

func TestGenerateThreeDayWindow(t *testing.T) {
	loc := tehranLoc(t)
	now := mondayNineAMTehran(t)

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          3,
		Location:            loc,
		MaxInterviewsPerDay: 99,
		WorkingHours: []WorkingHour{
			{DayOfWeek: 1, Start: clock(9, 0), End: clock(17, 0)},
			{DayOfWeek: 2, Start: clock(9, 0), End: clock(17, 0)},
			{DayOfWeek: 3, Start: clock(9, 0), End: clock(17, 0)},
			{DayOfWeek: 4, Start: clock(9, 0), End: clock(17, 0)},
			{DayOfWeek: 5, Start: clock(9, 0), End: clock(17, 0)},
		},
		DurationMinutes: 60,
		BufferMinutes:   0,
	})

	_, windowEnd := BookingWindow(now, loc, 3)
	for _, c := range candidates {
		if !c.StartAt.Before(windowEnd) {
			t.Fatalf("slot beyond 3-day window: %v", c.StartAt)
		}
	}
}

func TestBookingWindow(t *testing.T) {
	loc := tehranLoc(t)
	now := time.Date(2026, 7, 6, 15, 0, 0, 0, loc)

	start, end := BookingWindow(now, loc, 14)
	todayStart := startOfDay(now, loc)
	wantEnd := todayStart.AddDate(0, 0, 14)
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %v, want %v", end, wantEnd)
	}
	if start.Before(now.UTC()) {
		t.Fatalf("start %v before now", start)
	}
	if !start.Equal(now.UTC()) && !start.Equal(todayStart) {
		t.Fatalf("unexpected start %v", start)
	}
	_ = loc
}

func TestDurationFromClock(t *testing.T) {
	d, err := DurationFromClock("09:30")
	if err != nil {
		t.Fatal(err)
	}
	if d != clock(9, 30) {
		t.Fatalf("got %v", d)
	}

	d, err = DurationFromClock("09:30:15")
	if err != nil {
		t.Fatal(err)
	}
	if d != clock(9, 30)+15*time.Second {
		t.Fatalf("got %v", d)
	}
}
