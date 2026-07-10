package availability

import (
	"sort"
	"time"
)

const (
	minMaxPerDay         = 1
	maxMaxPerDay         = 50
	minBookingHorizon    = 1
	maxBookingHorizon    = 90
	defaultBookingHorizon = 14
)

type WorkingHourInput struct {
	DayOfWeek int16
	StartTime string
	EndTime   string
}

type BreakInput struct {
	DayOfWeek int16
	StartTime string
	EndTime   string
}

type TimeOffInput struct {
	StartAt string
	EndAt   string
}

type Input struct {
	Timezone            string
	MaxInterviewsPerDay int32
	BookingHorizonDays  int32
	WorkingHours        []WorkingHourInput
	Breaks              []BreakInput
	TimeOff             []TimeOffInput
}

type localInterval struct {
	start time.Duration
	end   time.Duration
}

func validateInput(in Input) (Input, *time.Location, []parsedTimeOff, error) {
	in.Timezone = trim(in.Timezone)
	if in.Timezone == "" {
		in.Timezone = "Asia/Tehran"
	}

	loc, err := time.LoadLocation(in.Timezone)
	if err != nil {
		return Input{}, nil, nil, ErrInvalidTimezone
	}

	if in.MaxInterviewsPerDay < minMaxPerDay || in.MaxInterviewsPerDay > maxMaxPerDay {
		return Input{}, nil, nil, ErrInvalidMaxPerDay
	}

	if in.BookingHorizonDays == 0 {
		in.BookingHorizonDays = defaultBookingHorizon
	}
	if in.BookingHorizonDays < minBookingHorizon || in.BookingHorizonDays > maxBookingHorizon {
		return Input{}, nil, nil, ErrInvalidBookingHorizon
	}

	hoursByDay, err := parseAndValidateHours(in.WorkingHours)
	if err != nil {
		return Input{}, nil, nil, err
	}

	if err := validateBreaks(in.Breaks, hoursByDay); err != nil {
		return Input{}, nil, nil, err
	}

	timeOff, err := parseAndValidateTimeOff(in.TimeOff)
	if err != nil {
		return Input{}, nil, nil, err
	}

	return in, loc, timeOff, nil
}

func parseAndValidateHours(hours []WorkingHourInput) (map[int16][]localInterval, error) {
	byDay := make(map[int16][]localInterval)
	for _, h := range hours {
		if h.DayOfWeek < 0 || h.DayOfWeek > 6 {
			return nil, ErrInvalidDayOfWeek
		}
		start, err := parseClock(h.StartTime)
		if err != nil {
			return nil, ErrInvalidTimeRange
		}
		end, err := parseClock(h.EndTime)
		if err != nil {
			return nil, ErrInvalidTimeRange
		}
		if end <= start {
			return nil, ErrInvalidTimeRange
		}
		byDay[h.DayOfWeek] = append(byDay[h.DayOfWeek], localInterval{start: start, end: end})
	}

	for day, intervals := range byDay {
		if hasOverlap(intervals) {
			return nil, ErrOverlappingHours
		}
		byDay[day] = intervals
	}
	return byDay, nil
}

func validateBreaks(breaks []BreakInput, hoursByDay map[int16][]localInterval) error {
	for _, b := range breaks {
		if b.DayOfWeek < 0 || b.DayOfWeek > 6 {
			return ErrInvalidDayOfWeek
		}
		start, err := parseClock(b.StartTime)
		if err != nil {
			return ErrInvalidTimeRange
		}
		end, err := parseClock(b.EndTime)
		if err != nil {
			return ErrInvalidTimeRange
		}
		if end <= start {
			return ErrInvalidTimeRange
		}

		dayHours := hoursByDay[b.DayOfWeek]
		if len(dayHours) == 0 {
			return ErrBreakOutsideHours
		}
		if !isSubset(localInterval{start: start, end: end}, dayHours) {
			return ErrBreakOutsideHours
		}
	}
	return nil
}

type parsedTimeOff struct {
	StartAt time.Time
	EndAt   time.Time
}

func parseAndValidateTimeOff(blocks []TimeOffInput) ([]parsedTimeOff, error) {
	out := make([]parsedTimeOff, 0, len(blocks))
	for _, b := range blocks {
		start, err := time.Parse(time.RFC3339, trim(b.StartAt))
		if err != nil {
			return nil, ErrInvalidTimeOffTime
		}
		end, err := time.Parse(time.RFC3339, trim(b.EndAt))
		if err != nil {
			return nil, ErrInvalidTimeOffTime
		}
		if !end.After(start) {
			return nil, ErrInvalidTimeOffRange
		}
		out = append(out, parsedTimeOff{StartAt: start.UTC(), EndAt: end.UTC()})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].StartAt.Before(out[j].StartAt)
	})
	for i := 1; i < len(out); i++ {
		if out[i].StartAt.Before(out[i-1].EndAt) {
			return nil, ErrOverlappingTimeOff
		}
	}
	return out, nil
}

func hasOverlap(intervals []localInterval) bool {
	if len(intervals) < 2 {
		return false
	}
	sorted := append([]localInterval(nil), intervals...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].start < sorted[j].start
	})
	for i := 1; i < len(sorted); i++ {
		if sorted[i].start < sorted[i-1].end {
			return true
		}
	}
	return false
}

func isSubset(block localInterval, hours []localInterval) bool {
	for _, h := range hours {
		if h.start <= block.start && h.end >= block.end {
			return true
		}
	}
	return false
}

func parseClock(value string) (time.Duration, error) {
	value = trim(value)
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

func trim(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n') {
		j--
	}
	return s[i:j]
}
