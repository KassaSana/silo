package stats

import (
	"database/sql"
	"testing"
	"time"

	"silo/app/store"

	_ "modernc.org/sqlite"
)

// calcStreak is the whole point of the streak widget on the dashboard.
// It has three boundary cases that users feel immediately:
//   1. Today-only — streak = 1.
//   2. Gap of one day — streak still counts (grace period).
//   3. Gap of two days — streak breaks (returns 0).

func TestCalcStreak_NoData(t *testing.T) {
	m := newTestManager(t)
	if got := m.calcStreak(); got != 0 {
		t.Errorf("empty db should have streak 0, got %d", got)
	}
}

func TestCalcStreak_TodayOnly(t *testing.T) {
	m := newTestManager(t)
	seedDay(t, m, 0)
	if got := m.calcStreak(); got != 1 {
		t.Errorf("today-only should give streak 1, got %d", got)
	}
}

func TestCalcStreak_ConsecutiveFive(t *testing.T) {
	m := newTestManager(t)
	for i := 0; i < 5; i++ {
		seedDay(t, m, -i)
	}
	if got := m.calcStreak(); got != 5 {
		t.Errorf("five consecutive days should give streak 5, got %d", got)
	}
}

func TestCalcStreak_GracePeriodOneDayGap(t *testing.T) {
	// User focused yesterday but not today (yet). Streak should still be
	// alive — breaks only after a full second day without focus.
	m := newTestManager(t)
	seedDay(t, m, -1)
	seedDay(t, m, -2)
	seedDay(t, m, -3)
	if got := m.calcStreak(); got != 3 {
		t.Errorf("yesterday + 2 days = 3 streak (grace day), got %d", got)
	}
}

func TestCalcStreak_BreaksAfterTwoDayGap(t *testing.T) {
	// Most recent session was TWO days ago — streak should be 0, regardless
	// of how long the earlier run was.
	m := newTestManager(t)
	seedDay(t, m, -2)
	seedDay(t, m, -3)
	seedDay(t, m, -4)
	if got := m.calcStreak(); got != 0 {
		t.Errorf("two-day gap should break streak, got %d", got)
	}
}

func TestCalcStreak_BrokenInMiddle(t *testing.T) {
	// Consecutive days 0, 1, 2, then skip 3, then 4, 5 earlier.
	// Streak counts from today backward and stops at the gap — answer is 3.
	m := newTestManager(t)
	for _, offset := range []int{0, -1, -2, -4, -5} {
		seedDay(t, m, offset)
	}
	if got := m.calcStreak(); got != 3 {
		t.Errorf("gap at day -3 should cap streak at 3, got %d", got)
	}
}

// GetSummary aggregates daily totals. Unit-test the math without worrying
// about sessions/commit messages — those are covered by session integration.

func TestGetSummary_EmptyDB(t *testing.T) {
	m := newTestManager(t)
	s, err := m.GetSummary()
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if s.TodayMinutes != 0 || s.WeekMinutes != 0 || s.StreakDays != 0 ||
		s.TotalSessions != 0 || s.TotalFocusMinutes != 0 {
		t.Errorf("empty summary should be all zero, got %+v", s)
	}
}

func TestGetSummary_TodayAndWeek(t *testing.T) {
	m := newTestManager(t)
	// Seed: 60 min today, 30 min yesterday, 10 min 10 days ago (outside week)
	seedDayWithFocus(t, m, 0, 60*60)
	seedDayWithFocus(t, m, -1, 30*60)
	seedDayWithFocus(t, m, -10, 10*60)

	s, err := m.GetSummary()
	if err != nil {
		t.Fatal(err)
	}
	if s.TodayMinutes != 60 {
		t.Errorf("today: got %d, want 60", s.TodayMinutes)
	}
	if s.WeekMinutes != 90 {
		t.Errorf("week: got %d, want 90 (today + yesterday)", s.WeekMinutes)
	}
	if s.TotalFocusMinutes != 100 {
		t.Errorf("lifetime: got %d, want 100", s.TotalFocusMinutes)
	}
}

// ── test helpers ──

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	// Hand-roll just the daily_stats + sessions schema we need — we don't
	// want to depend on store.migrate() because that runs against the real
	// Store and exercises the full DB lifecycle. A minimal schema keeps this
	// test focused on the stats logic.
	_, err = db.Exec(`
		CREATE TABLE daily_stats (
			date TEXT PRIMARY KEY,
			total_focus_seconds INTEGER DEFAULT 0,
			session_count INTEGER DEFAULT 0,
			blocked_attempt_count INTEGER DEFAULT 0
		);
		CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			workspace_id TEXT,
			task_description TEXT,
			commit_message TEXT,
			duration_actual INTEGER,
			duration_planned INTEGER,
			started_at TEXT,
			completed_at TEXT,
			status TEXT,
			breach_attempts INTEGER
		);
		CREATE TABLE workspaces (
			id TEXT PRIMARY KEY,
			name TEXT
		);`)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewManager(&store.Store{DB: db})
}

// seedDay inserts a daily_stats row for today+offsetDays with a default
// focus amount. offset is negative for past days (e.g. -1 = yesterday).
func seedDay(t *testing.T, m *Manager, offsetDays int) {
	t.Helper()
	seedDayWithFocus(t, m, offsetDays, 1800) // 30 min
}

func seedDayWithFocus(t *testing.T, m *Manager, offsetDays, focusSecs int) {
	t.Helper()
	date := time.Now().AddDate(0, 0, offsetDays).Format("2006-01-02")
	_, err := m.store.DB.Exec(
		`INSERT INTO daily_stats (date, total_focus_seconds, session_count, blocked_attempt_count)
		 VALUES (?, ?, 1, 0)`,
		date, focusSecs,
	)
	if err != nil {
		t.Fatalf("seed day %d: %v", offsetDays, err)
	}
}
