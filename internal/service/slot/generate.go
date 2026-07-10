package slot

import (
	"time"
)

const DefaultWindowDays = 14

// SlotCandidate is a generated bookable window in UTC.
type SlotCandidate struct {
	StartAt time.Time
	EndAt   time.Time
}

// WorkingHour is a recurring local-time window on a weekday (0=Sunday).
type WorkingHour struct {
	DayOfWeek int16
	Start     time.Duration
	End       time.Duration
}

// Break is a recurring local-time exclusion on a weekday.
type Break struct {
	DayOfWeek int16
	Start     time.Duration
	End       time.Duration
}

// TimeOff is an absolute UTC block [StartAt, EndAt).
type TimeOff struct {
	StartAt time.Time
	EndAt   time.Time
}

type localInterval struct {
	start time.Duration
	end   time.Duration
}

// GenerateParams configures slot generation for one interview type.
type GenerateParams struct {
	Now                 time.Time
	WindowDays          int
	Location            *time.Location
	MaxInterviewsPerDay int32
	WorkingHours        []WorkingHour
	Breaks              []Break
	TimeOff             []TimeOff
	DurationMinutes     int32
	BufferMinutes       int32
	// SlotsPerLocalDay returns existing slot count for a local calendar day (booked + unbooked).
	SlotsPerLocalDay func(localDay time.Time) int32
	// OccupiedStarts are UTC slot starts that must not be re-inserted (e.g. booked).
	OccupiedStarts map[time.Time]struct{}
}

// Generate produces slot candidates within the rolling window.
func Generate(p GenerateParams) []SlotCandidate {
	if p.WindowDays <= 0 {
		p.WindowDays = DefaultWindowDays
	}
	if p.Location == nil {
		p.Location = time.UTC
	}
	if p.MaxInterviewsPerDay <= 0 || p.DurationMinutes <= 0 {
		return nil
	}

	windowStart, windowEnd := regenWindow(p.Now, p.Location, p.WindowDays)
	if !windowStart.Before(windowEnd) {
		return nil
	}

	hoursByDay := groupWorkingHours(p.WorkingHours)
	breaksByDay := groupBreaks(p.Breaks)
	step := time.Duration(p.DurationMinutes+p.BufferMinutes) * time.Minute
	slotDur := time.Duration(p.DurationMinutes) * time.Minute

	var out []SlotCandidate
	for day := startOfDay(windowStart, p.Location); day.Before(windowEnd); day = day.AddDate(0, 0, 1) {
		weekday := int16(day.In(p.Location).Weekday())
		free := hoursByDay[weekday]
		if len(free) == 0 {
			continue
		}

		free = subtractIntervals(free, breaksByDay[weekday])
		free = subtractIntervals(free, timeOffIntervalsForDay(day, p.Location, p.TimeOff))
		if len(free) == 0 {
			continue
		}

		slotsToday := int32(0)
		if p.SlotsPerLocalDay != nil {
			slotsToday = p.SlotsPerLocalDay(day)
		}

		for _, interval := range free {
			for t := interval.start; t+slotDur <= interval.end; t += step {
				startUTC := localTimeOnDay(day, t, p.Location)
				endUTC := startUTC.Add(slotDur)
				if startUTC.Before(windowStart) || !startUTC.Before(windowEnd) {
					continue
				}
				if p.OccupiedStarts != nil {
					if _, ok := p.OccupiedStarts[startUTC.UTC()]; ok {
						continue
					}
				}
				if slotsToday >= p.MaxInterviewsPerDay {
					break
				}
				out = append(out, SlotCandidate{StartAt: startUTC, EndAt: endUTC})
				slotsToday++
			}
			if slotsToday >= p.MaxInterviewsPerDay {
				break
			}
		}
	}

	return out
}

func regenWindow(now time.Time, loc *time.Location, days int) (start, end time.Time) {
	return BookingWindow(now, loc, days)
}

// BookingWindow returns the bookable UTC range [start, end) for a horizon of calendar days.
func BookingWindow(now time.Time, loc *time.Location, days int) (start, end time.Time) {
	if days <= 0 {
		days = DefaultWindowDays
	}
	todayStart := startOfDay(now, loc)
	start = maxTime(now.UTC(), todayStart)
	end = todayStart.AddDate(0, 0, days)
	return start, end
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func localTimeOnDay(day time.Time, offset time.Duration, loc *time.Location) time.Time {
	return startOfDay(day, loc).Add(offset)
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func groupWorkingHours(hours []WorkingHour) map[int16][]localInterval {
	out := make(map[int16][]localInterval)
	for _, h := range hours {
		if h.End <= h.Start {
			continue
		}
		out[h.DayOfWeek] = append(out[h.DayOfWeek], localInterval{start: h.Start, end: h.End})
	}
	return out
}

func groupBreaks(breaks []Break) map[int16][]localInterval {
	out := make(map[int16][]localInterval)
	for _, b := range breaks {
		if b.End <= b.Start {
			continue
		}
		out[b.DayOfWeek] = append(out[b.DayOfWeek], localInterval{start: b.Start, end: b.End})
	}
	return out
}

func subtractIntervals(base, cuts []localInterval) []localInterval {
	if len(base) == 0 || len(cuts) == 0 {
		return base
	}
	result := base
	for _, cut := range cuts {
		var next []localInterval
		for _, iv := range result {
			next = append(next, subtractOne(iv, cut)...)
		}
		result = next
	}
	return result
}

func subtractOne(base, cut localInterval) []localInterval {
	if cut.end <= base.start || cut.start >= base.end {
		return []localInterval{base}
	}
	var out []localInterval
	if cut.start > base.start {
		out = append(out, localInterval{start: base.start, end: cut.start})
	}
	if cut.end < base.end {
		out = append(out, localInterval{start: cut.end, end: base.end})
	}
	return out
}

func timeOffIntervalsForDay(day time.Time, loc *time.Location, blocks []TimeOff) []localInterval {
	dayStart := startOfDay(day, loc)
	dayEnd := dayStart.Add(24 * time.Hour)

	var cuts []localInterval
	for _, b := range blocks {
		if !b.EndAt.After(b.StartAt) {
			continue
		}
		overlapStart := maxTime(b.StartAt, dayStart)
		overlapEnd := minTime(b.EndAt, dayEnd)
		if overlapStart.Before(overlapEnd) {
			cuts = append(cuts, localInterval{
				start: overlapStart.Sub(dayStart),
				end:   overlapEnd.Sub(dayStart),
			})
		}
	}
	return cuts
}

// DurationFromClock parses "HH:MM" or "HH:MM:SS" into a duration from local midnight.
func DurationFromClock(value string) (time.Duration, error) {
	var parsed time.Time
	var err error
	for _, layout := range []string{"15:04:05", "15:04"} {
		parsed, err = time.Parse(layout, value)
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, err
	}
	return time.Duration(parsed.Hour())*time.Hour +
		time.Duration(parsed.Minute())*time.Minute +
		time.Duration(parsed.Second())*time.Second, nil
}

func durationFromPgMicros(micros int64) time.Duration {
	return time.Duration(micros) * time.Microsecond
}
