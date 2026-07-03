package slot

import (
	"context"
	"errors"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	q   *db.Queries
	Now func() time.Time
}

func NewService(q *db.Queries) *Service {
	return &Service{
		q:   q,
		Now: time.Now,
	}
}

func (s *Service) queries(q *db.Queries) *db.Queries {
	if q != nil {
		return q
	}
	return s.q
}

// RegenerateForOrg rebuilds slots for every interview type on the org.
func (s *Service) RegenerateForOrg(ctx context.Context, q *db.Queries, orgID pgtype.UUID) error {
	queries := s.queries(q)

	types, err := queries.ListInterviewTypesByOrg(ctx, orgID)
	if err != nil {
		return err
	}

	avail, err := s.loadAvailability(ctx, queries, orgID)
	if err != nil {
		if errors.Is(err, ErrAvailabilityNotConfigured) {
			return nil
		}
		return err
	}

	for _, it := range types {
		if err := s.regenerateForType(ctx, queries, orgID, it, avail); err != nil {
			return err
		}
	}
	return nil
}

// RegenerateForType rebuilds unbooked slots for one interview type.
func (s *Service) RegenerateForType(
	ctx context.Context,
	q *db.Queries,
	orgID, typeID pgtype.UUID,
	durationMinutes, bufferMinutes int32,
) error {
	queries := s.queries(q)

	it, err := queries.GetInterviewTypeByID(ctx, typeID)
	if err != nil {
		return err
	}
	if !it.OrganizationID.Valid || !orgID.Valid || it.OrganizationID.Bytes != orgID.Bytes {
		return ErrAvailabilityNotConfigured
	}

	avail, err := s.loadAvailability(ctx, queries, orgID)
	if err != nil {
		if errors.Is(err, ErrAvailabilityNotConfigured) {
			return nil
		}
		return err
	}

	it.DurationMinutes = durationMinutes
	it.BufferMinutes = bufferMinutes
	return s.regenerateForType(ctx, queries, orgID, it, avail)
}

type availabilitySnapshot struct {
	loc     *time.Location
	maxDay  int32
	hours   []WorkingHour
	breaks  []Break
	timeOff []TimeOff
}

func (s *Service) loadAvailability(ctx context.Context, q *db.Queries, orgID pgtype.UUID) (availabilitySnapshot, error) {
	settings, err := q.GetAvailabilitySettingsByOrg(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return availabilitySnapshot{}, ErrAvailabilityNotConfigured
		}
		return availabilitySnapshot{}, err
	}

	loc, err := time.LoadLocation(settings.Timezone)
	if err != nil {
		return availabilitySnapshot{}, ErrInvalidTimezone
	}

	wh, err := q.ListWorkingHoursByOrg(ctx, orgID)
	if err != nil {
		return availabilitySnapshot{}, err
	}
	br, err := q.ListBreaksByOrg(ctx, orgID)
	if err != nil {
		return availabilitySnapshot{}, err
	}
	to, err := q.ListTimeOffByOrg(ctx, orgID)
	if err != nil {
		return availabilitySnapshot{}, err
	}

	return availabilitySnapshot{
		loc:     loc,
		maxDay:  settings.MaxInterviewsPerDay,
		hours:   workingHoursFromDB(wh),
		breaks:  breaksFromDB(br),
		timeOff: timeOffFromDB(to, loc),
	}, nil
}

func (s *Service) regenerateForType(
	ctx context.Context,
	q *db.Queries,
	orgID pgtype.UUID,
	it db.InterviewType,
	avail availabilitySnapshot,
) error {
	now := s.Now()
	windowStart, windowEnd := regenWindow(now, avail.loc, DefaultWindowDays)

	if err := q.DeleteUnbookedSlotsByTypeInWindow(ctx, db.DeleteUnbookedSlotsByTypeInWindowParams{
		InterviewTypeID: it.ID,
		StartAt:         timestamptz(windowStart),
		StartAt_2:       timestamptz(windowEnd),
	}); err != nil {
		return err
	}

	remaining, err := q.ListSlotsByTypeInWindow(ctx, db.ListSlotsByTypeInWindowParams{
		InterviewTypeID: it.ID,
		StartAt:         timestamptz(windowStart),
		StartAt_2:       timestamptz(windowEnd),
	})
	if err != nil {
		return err
	}

	occupied := make(map[time.Time]struct{}, len(remaining))
	for _, slot := range remaining {
		occupied[slot.StartAt.Time.UTC()] = struct{}{}
	}

	slotsPerDay := func(localDay time.Time) int32 {
		dayStart, dayEnd := localDayBounds(localDay, avail.loc)
		count, err := q.CountSlotsByOrgOnLocalDay(ctx, db.CountSlotsByOrgOnLocalDayParams{
			OrganizationID: orgID,
			StartAt:        timestamptz(dayStart),
			StartAt_2:      timestamptz(dayEnd),
		})
		if err != nil {
			return avail.maxDay
		}
		return count
	}

	candidates := Generate(GenerateParams{
		Now:                 now,
		WindowDays:          DefaultWindowDays,
		Location:            avail.loc,
		MaxInterviewsPerDay: avail.maxDay,
		WorkingHours:        avail.hours,
		Breaks:              avail.breaks,
		TimeOff:             avail.timeOff,
		DurationMinutes:     it.DurationMinutes,
		BufferMinutes:       it.BufferMinutes,
		SlotsPerLocalDay:    slotsPerDay,
		OccupiedStarts:      occupied,
	})
	if len(candidates) == 0 {
		return nil
	}

	rows := make([]db.BulkInsertSlotsParams, 0, len(candidates))
	for _, c := range candidates {
		rows = append(rows, db.BulkInsertSlotsParams{
			OrganizationID:  orgID,
			InterviewTypeID: it.ID,
			StartAt:         timestamptz(c.StartAt),
			EndAt:           timestamptz(c.EndAt),
		})
	}

	_, err = q.BulkInsertSlots(ctx, rows)
	return err
}

func workingHoursFromDB(rows []db.AvailabilityWorkingHour) []WorkingHour {
	out := make([]WorkingHour, 0, len(rows))
	for _, r := range rows {
		out = append(out, WorkingHour{
			DayOfWeek: r.DayOfWeek,
			Start:     durationFromPgMicros(r.StartTime.Microseconds),
			End:       durationFromPgMicros(r.EndTime.Microseconds),
		})
	}
	return out
}

func breaksFromDB(rows []db.AvailabilityBreak) []Break {
	out := make([]Break, 0, len(rows))
	for _, r := range rows {
		out = append(out, Break{
			DayOfWeek: r.DayOfWeek,
			Start:     durationFromPgMicros(r.StartTime.Microseconds),
			End:       durationFromPgMicros(r.EndTime.Microseconds),
		})
	}
	return out
}

func timeOffFromDB(rows []db.AvailabilityTimeOff, loc *time.Location) []TimeOff {
	out := make([]TimeOff, 0, len(rows))
	for _, r := range rows {
		if !r.StartDate.Valid || !r.EndAt.Valid {
			continue
		}
		start := dateAtMidnight(r.StartDate, loc)
		out = append(out, TimeOff{
			StartAt: start,
			EndAt:   r.EndAt.Time,
		})
	}
	return out
}

func dateAtMidnight(d pgtype.Date, loc *time.Location) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	t := d.Time
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func localDayBounds(day time.Time, loc *time.Location) (time.Time, time.Time) {
	start := startOfDay(day, loc)
	return start, start.Add(24 * time.Hour)
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
