package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	authsvc 	"github.com/AmirAbaris/weeto-backend/internal/service/auth"
	availabilitysvc "github.com/AmirAbaris/weeto-backend/internal/service/availability"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	interviewtypesvc "github.com/AmirAbaris/weeto-backend/internal/service/interviewtype"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	slotsvc "github.com/AmirAbaris/weeto-backend/internal/service/slot"
	"github.com/AmirAbaris/weeto-backend/internal/platform/crypto"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var (
	migrateOnce       sync.Once
	migrateErr        error
	migrationsDirPath string
)

func init() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		migrationsDirPath = "db/migrations"
		return
	}
	migrationsDirPath = filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "db", "migrations"))
}

func testDatabaseURL() string {
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "postgres://weeto:weeto@localhost:5432/weeto_test?sslmode=disable"
}

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	dsn := testDatabaseURL()

	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Skipf("integration database unavailable: %v", err)
	}

	migrateOnce.Do(func() {
		sqlDB, err := sql.Open("pgx", dsn)
		if err != nil {
			migrateErr = fmt.Errorf("open sql db: %w", err)
			return
		}
		defer sqlDB.Close()

		if err := goose.SetDialect("postgres"); err != nil {
			migrateErr = err
			return
		}
		migrateErr = goose.Up(sqlDB, migrationsDirPath)
	})

	if migrateErr != nil {
		t.Fatalf("migrate: %v", migrateErr)
	}

	TruncateAll(t, pool)
	return pool
}

func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE TABLE
			notification_outbox,
			booking,
			slots,
			availability_time_off,
			availability_breaks,
			availability_working_hours,
			availability_settings,
			interview_type,
			organization,
			refresh_tokens,
			users
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

type TestEnv struct {
	T          *testing.T
	Ctx        context.Context
	Pool       *pgxpool.Pool
	Queries    *db.Queries
	OrgID      pgtype.UUID
	OrgSlug    string
	UserID     pgtype.UUID
	Now        time.Time
	SlotSvc    *slotsvc.Service
	AvailSvc   *availabilitysvc.Service
	ITSvc      *interviewtypesvc.Service
	BookingSvc *bookingsvc.Service
	Calendar   *MockCalendarClient
}

func NewTestEnv(t *testing.T, now time.Time) *TestEnv {
	return NewTestEnvWithCalendar(t, now, &MockCalendarClient{})
}

func NewTestEnvWithCalendar(t *testing.T, now time.Time, calendar *MockCalendarClient) *TestEnv {
	t.Helper()

	pool := SetupTestDB(t)
	queries := db.New(pool)
	cfg := &config.Config{JWTSecret: "integration-test-secret"}

	orgSvc := orgsvc.NewService(queries, cfg)
	slotSvc := slotsvc.NewService(queries)
	slotSvc.Now = func() time.Time { return now }
	availSvc := availabilitysvc.NewService(pool, queries, orgSvc, slotSvc)
	itSvc := interviewtypesvc.NewService(queries, orgSvc, slotSvc)
	bookingSvc := bookingsvc.NewService(pool, queries, orgSvc, slotSvc, calendar)

	userID := seedUser(t, queries)
	orgID, orgSlug := seedOrg(t, queries, userID)

	return &TestEnv{
		T:          t,
		Ctx:        context.Background(),
		Pool:       pool,
		Queries:    queries,
		OrgID:      orgID,
		OrgSlug:    orgSlug,
		UserID:     userID,
		Now:        now,
		SlotSvc:    slotSvc,
		AvailSvc:   availSvc,
		ITSvc:      itSvc,
		BookingSvc: bookingSvc,
		Calendar:   calendar,
	}
}

func seedUser(t *testing.T, q *db.Queries) pgtype.UUID {
	t.Helper()
	hash, err := authsvc.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	user, err := q.CreateUser(context.Background(), db.CreateUserParams{
		Email:        fmt.Sprintf("user-%d@test.local", time.Now().UnixNano()),
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatal(err)
	}
	return user.ID
}

func seedOrg(t *testing.T, q *db.Queries, ownerID pgtype.UUID) (pgtype.UUID, string) {
	t.Helper()
	slug := fmt.Sprintf("test-org-%d", time.Now().UnixNano())
	org, err := q.CreateOrganization(context.Background(), db.CreateOrganizationParams{
		Name:    "Test Org",
		Slug:    slug,
		OwnerID: ownerID,
		Plan:    db.PlanTypeFree,
	})
	if err != nil {
		t.Fatal(err)
	}
	return org.ID, slug
}

func (e *TestEnv) UpsertAvailability(in availabilitysvc.Input) {
	e.T.Helper()
	if _, err := e.AvailSvc.Upsert(e.Ctx, e.UserID, in); err != nil {
		e.T.Fatalf("upsert availability: %v", err)
	}
}

func (e *TestEnv) CreateInterviewType(duration, buffer int32) db.InterviewType {
	e.T.Helper()
	slug := fmt.Sprintf("type-%d", time.Now().UnixNano())
	it, err := e.Queries.CreateInterviewType(e.Ctx, db.CreateInterviewTypeParams{
		OrganizationID:  e.OrgID,
		Title:           "Interview",
		Slug:            slug,
		DurationMinutes: duration,
		BufferMinutes:   buffer,
		MeetingProvider: db.MeetingProviderCustomUrl,
		MeetingUrl:      pgtype.Text{String: "https://example.com/meet", Valid: true},
	})
	if err != nil {
		e.T.Fatal(err)
	}
	if err := e.SlotSvc.RegenerateForType(e.Ctx, nil, e.OrgID, it.ID, duration, buffer); err != nil {
		e.T.Fatalf("regenerate slots: %v", err)
	}
	return it
}

var testEncryptionKey = []byte("01234567890123456789012345678901")

func (e *TestEnv) ConnectGoogle() {
	e.T.Helper()
	encrypted, err := crypto.Encrypt([]byte("test-refresh-token"), testEncryptionKey)
	if err != nil {
		e.T.Fatal(err)
	}
	if err := e.Queries.SetUserGoogleConnection(e.Ctx, db.SetUserGoogleConnectionParams{
		ID:                 e.UserID,
		GoogleID:           pgtype.Text{String: "google-user-123", Valid: true},
		GoogleRefreshToken: pgtype.Text{String: encrypted, Valid: true},
	}); err != nil {
		e.T.Fatal(err)
	}
}

func (e *TestEnv) CreateGoogleMeetInterviewType(duration, buffer int32) db.InterviewType {
	e.T.Helper()
	e.ConnectGoogle()
	slug := fmt.Sprintf("meet-type-%d", time.Now().UnixNano())
	it, err := e.Queries.CreateInterviewType(e.Ctx, db.CreateInterviewTypeParams{
		OrganizationID:  e.OrgID,
		Title:           "Google Meet Interview",
		Slug:            slug,
		DurationMinutes: duration,
		BufferMinutes:   buffer,
		MeetingProvider: db.MeetingProviderGoogleMeet,
	})
	if err != nil {
		e.T.Fatal(err)
	}
	if err := e.SlotSvc.RegenerateForType(e.Ctx, nil, e.OrgID, it.ID, duration, buffer); err != nil {
		e.T.Fatalf("regenerate slots: %v", err)
	}
	return it
}

func (e *TestEnv) SetMeetLinksUsed(count int32) {
	e.T.Helper()
	_, err := e.Pool.Exec(e.Ctx, `UPDATE organization SET meet_links_used = $2 WHERE id = $1`, e.OrgID, count)
	if err != nil {
		e.T.Fatal(err)
	}
}

func (e *TestEnv) ListSlots(typeID pgtype.UUID) []db.Slot {
	e.T.Helper()
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		e.T.Fatal(err)
	}
	today := time.Date(e.Now.In(loc).Year(), e.Now.In(loc).Month(), e.Now.In(loc).Day(), 0, 0, 0, 0, loc)
	end := today.AddDate(0, 0, slotsvc.DefaultWindowDays)

	slots, err := e.Queries.ListSlotsByTypeInWindow(e.Ctx, db.ListSlotsByTypeInWindowParams{
		InterviewTypeID: typeID,
		StartAt:         pgTimestamptz(today),
		StartAt_2:       pgTimestamptz(end),
	})
	if err != nil {
		e.T.Fatal(err)
	}
	return slots
}

func (e *TestEnv) SlotsOnLocalDay(typeID pgtype.UUID, day time.Time, loc *time.Location) []db.Slot {
	e.T.Helper()
	dayStart := time.Date(day.In(loc).Year(), day.In(loc).Month(), day.In(loc).Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24 * time.Hour)
	slots, err := e.Queries.ListSlotsByTypeInWindow(e.Ctx, db.ListSlotsByTypeInWindowParams{
		InterviewTypeID: typeID,
		StartAt:         pgTimestamptz(dayStart),
		StartAt_2:       pgTimestamptz(dayEnd),
	})
	if err != nil {
		e.T.Fatal(err)
	}
	return slots
}

func (e *TestEnv) FirstAvailableSlot(it db.InterviewType) db.Slot {
	e.T.Helper()
	for _, slot := range e.ListSlots(it.ID) {
		if !slot.Booked {
			return slot
		}
	}
	e.T.Fatal("no available slots")
	return db.Slot{}
}

func (e *TestEnv) BookSlot(slotID pgtype.UUID) {
	e.T.Helper()
	if err := e.Queries.SetSlotBooked(e.Ctx, db.SetSlotBookedParams{
		ID:     slotID,
		Booked: true,
	}); err != nil {
		e.T.Fatal(err)
	}
}

func pgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func IssueTestToken(t *testing.T, userID pgtype.UUID) string {
	t.Helper()
	token, err := authsvc.IssueAccessToken(userID, "integration-test-secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
