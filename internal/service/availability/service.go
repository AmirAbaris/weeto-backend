package availability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/db"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	slotsvc "github.com/AmirAbaris/weeto-backend/internal/service/slot"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool    *pgxpool.Pool
	q       *db.Queries
	orgSvc  *orgsvc.Service
	slotSvc *slotsvc.Service
}

func NewService(pool *pgxpool.Pool, q *db.Queries, orgSvc *orgsvc.Service, slotSvc *slotsvc.Service) *Service {
	return &Service{
		pool:    pool,
		q:       q,
		orgSvc:  orgSvc,
		slotSvc: slotSvc,
	}
}

type View struct {
	Timezone            string
	MaxInterviewsPerDay int32
	WorkingHours        []WorkingHourView
	Breaks              []BreakView
	TimeOff             []TimeOffView
	UpdatedAt           time.Time
}

type WorkingHourView struct {
	ID         string
	DayOfWeek  int16
	StartTime  string
	EndTime    string
	CreatedAt  time.Time
}

type BreakView struct {
	ID        string
	DayOfWeek int16
	StartTime string
	EndTime   string
}

type TimeOffView struct {
	ID      string
	StartAt time.Time
	EndAt   time.Time
}

func (s *Service) Upsert(ctx context.Context, ownerID pgtype.UUID, in Input) (View, error) {
	if !ownerID.Valid {
		return View{}, ErrForbidden
	}

	in, _, timeOff, err := validateInput(in)
	if err != nil {
		return View{}, err
	}

	org, err := s.orgSvc.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, orgsvc.ErrOrgNotFound) {
			return View{}, ErrOrgRequired
		}
		return View{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return View{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	if err := qtx.UpsertAvailabilitySettings(ctx, db.UpsertAvailabilitySettingsParams{
		OrganizationID:      org.ID,
		Timezone:            in.Timezone,
		MaxInterviewsPerDay: in.MaxInterviewsPerDay,
	}); err != nil {
		return View{}, err
	}

	if err := qtx.DeleteWorkingHoursByOrg(ctx, org.ID); err != nil {
		return View{}, err
	}
	for _, h := range in.WorkingHours {
		start, err := parseClock(h.StartTime)
		if err != nil {
			return View{}, ErrInvalidTimeRange
		}
		end, err := parseClock(h.EndTime)
		if err != nil {
			return View{}, ErrInvalidTimeRange
		}
		if err := qtx.InsertWorkingHour(ctx, db.InsertWorkingHourParams{
			OrganizationID: org.ID,
			DayOfWeek:      h.DayOfWeek,
			StartTime:      pgTime(start),
			EndTime:        pgTime(end),
		}); err != nil {
			return View{}, err
		}
	}

	if err := qtx.DeleteBreaksByOrg(ctx, org.ID); err != nil {
		return View{}, err
	}
	for _, b := range in.Breaks {
		start, err := parseClock(b.StartTime)
		if err != nil {
			return View{}, ErrInvalidTimeRange
		}
		end, err := parseClock(b.EndTime)
		if err != nil {
			return View{}, ErrInvalidTimeRange
		}
		if err := qtx.InsertBreak(ctx, db.InsertBreakParams{
			OrganizationID: org.ID,
			DayOfWeek:      b.DayOfWeek,
			StartTime:      pgTime(start),
			EndTime:        pgTime(end),
		}); err != nil {
			return View{}, err
		}
	}

	if err := qtx.DeleteTimeOffByOrg(ctx, org.ID); err != nil {
		return View{}, err
	}
	for _, block := range timeOff {
		if _, err := qtx.InsertTimeOff(ctx, db.InsertTimeOffParams{
			OrganizationID: org.ID,
			StartAt:        timestamptz(block.StartAt),
			EndAt:          timestamptz(block.EndAt),
		}); err != nil {
			return View{}, err
		}
	}

	if err := s.slotSvc.RegenerateForOrg(ctx, qtx, org.ID); err != nil {
		return View{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return View{}, err
	}

	return s.Get(ctx, ownerID)
}

func (s *Service) Get(ctx context.Context, ownerID pgtype.UUID) (View, error) {
	if !ownerID.Valid {
		return View{}, ErrForbidden
	}

	org, err := s.orgSvc.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, orgsvc.ErrOrgNotFound) {
			return View{}, ErrOrgRequired
		}
		return View{}, err
	}

	settings, err := s.q.GetAvailabilitySettingsByOrg(ctx, org.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return View{}, ErrNotFound
		}
		return View{}, err
	}

	wh, err := s.q.ListWorkingHoursByOrg(ctx, org.ID)
	if err != nil {
		return View{}, err
	}
	br, err := s.q.ListBreaksByOrg(ctx, org.ID)
	if err != nil {
		return View{}, err
	}
	to, err := s.q.ListTimeOffByOrg(ctx, org.ID)
	if err != nil {
		return View{}, err
	}

	return buildView(settings, wh, br, to), nil
}

func buildView(
	settings db.AvailabilitySetting,
	wh []db.AvailabilityWorkingHour,
	br []db.AvailabilityBreak,
	to []db.AvailabilityTimeOff,
) View {
	view := View{
		Timezone:            settings.Timezone,
		MaxInterviewsPerDay: settings.MaxInterviewsPerDay,
		WorkingHours:        make([]WorkingHourView, 0, len(wh)),
		Breaks:              make([]BreakView, 0, len(br)),
		TimeOff:             make([]TimeOffView, 0, len(to)),
	}
	if settings.UpdatedAt.Valid {
		view.UpdatedAt = settings.UpdatedAt.Time
	}

	for _, row := range wh {
		item := WorkingHourView{
			DayOfWeek: row.DayOfWeek,
			StartTime: formatDuration(durationFromPgTime(row.StartTime)),
			EndTime:   formatDuration(durationFromPgTime(row.EndTime)),
		}
		if row.ID.Valid {
			item.ID = row.ID.String()
		}
		if row.CreatedAt.Valid {
			item.CreatedAt = row.CreatedAt.Time
		}
		view.WorkingHours = append(view.WorkingHours, item)
	}

	for _, row := range br {
		item := BreakView{
			DayOfWeek: row.DayOfWeek,
			StartTime: formatDuration(durationFromPgTime(row.StartTime)),
			EndTime:   formatDuration(durationFromPgTime(row.EndTime)),
		}
		if row.ID.Valid {
			item.ID = row.ID.String()
		}
		view.Breaks = append(view.Breaks, item)
	}

	for _, row := range to {
		item := TimeOffView{}
		if row.ID.Valid {
			item.ID = row.ID.String()
		}
		if row.StartAt.Valid {
			item.StartAt = row.StartAt.Time
		}
		if row.EndAt.Valid {
			item.EndAt = row.EndAt.Time
		}
		view.TimeOff = append(view.TimeOff, item)
	}

	return view
}

func pgTime(d time.Duration) pgtype.Time {
	return pgtype.Time{
		Microseconds: int64(d / time.Microsecond),
		Valid:        true,
	}
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func durationFromPgTime(t pgtype.Time) time.Duration {
	return time.Duration(t.Microseconds) * time.Microsecond
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%02d:%02d", h, m)
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
