package stats

/*
 * stats.go — Focus history queries and aggregations.
 *
 * CONCEPT: The blocker/session packages WRITE session data to SQLite.
 * This package READS it back for display and export.
 *
 * Three kinds of query:
 *   - Summary:   aggregate totals (today, this week, streak, lifetime)
 *   - Recent:    most recent N sessions joined with workspace names
 *   - Export:    raw JSON dump of everything (user-owned data)
 *
 * WHY separate package? Clear separation of concerns. session/ owns
 * writes (mutations during the lifecycle), stats/ owns reads (display).
 * Both talk to the same SQLite database through the shared store.
 */

import (
	"encoding/json"
	"fmt"
	"time"

	"silo/app/store"
)

// Summary is the headline stats shown on Dashboard + Stats screens.
type Summary struct {
	TodayMinutes      int `json:"today_minutes"`
	WeekMinutes       int `json:"week_minutes"`
	StreakDays        int `json:"streak_days"`
	TotalSessions     int `json:"total_sessions"`
	TotalFocusMinutes int `json:"total_focus_minutes"`
}

// Session is a single completed session joined with its workspace name.
// The frontend displays these in the "recent sessions" list.
type Session struct {
	ID              string `json:"id"`
	WorkspaceID     string `json:"workspace_id"`
	WorkspaceName   string `json:"workspace_name"`
	TaskDescription string `json:"task_description"`
	CommitMessage   string `json:"commit_message"`
	DurationActual  int    `json:"duration_actual"` // seconds
	DurationPlanned int    `json:"duration_planned"`
	StartedAt       string `json:"started_at"`
	CompletedAt     string `json:"completed_at"`
	Status          string `json:"status"`
	BreachAttempts  int    `json:"breach_attempts"`
}

// Manager owns the stats queries.
type Manager struct {
	store *store.Store
}

// NewManager creates a stats manager.
func NewManager(s *store.Store) *Manager {
	return &Manager{store: s}
}

// GetSummary computes headline focus metrics in a single round-trip pattern.
// All times are derived from daily_stats (populated at session completion).
func (m *Manager) GetSummary() (Summary, error) {
	today := time.Now().Format("2006-01-02")

	var s Summary

	// Today
	row := m.store.DB.QueryRow(
		`SELECT COALESCE(total_focus_seconds, 0) FROM daily_stats WHERE date = ?`, today,
	)
	var todaySecs int
	_ = row.Scan(&todaySecs)
	s.TodayMinutes = todaySecs / 60

	// This week: last 7 days including today
	weekStart := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	row = m.store.DB.QueryRow(
		`SELECT COALESCE(SUM(total_focus_seconds), 0)
		 FROM daily_stats WHERE date >= ?`, weekStart,
	)
	var weekSecs int
	_ = row.Scan(&weekSecs)
	s.WeekMinutes = weekSecs / 60

	// Streak: consecutive days with session_count > 0 working backward from today
	s.StreakDays = m.calcStreak()

	// Lifetime totals
	row = m.store.DB.QueryRow(
		`SELECT COALESCE(SUM(session_count), 0), COALESCE(SUM(total_focus_seconds), 0)
		 FROM daily_stats`,
	)
	var totalSess, totalSecs int
	_ = row.Scan(&totalSess, &totalSecs)
	s.TotalSessions = totalSess
	s.TotalFocusMinutes = totalSecs / 60

	return s, nil
}

// calcStreak walks backward from today, counting consecutive days
// where a session was logged. A gap of one missing day ends the streak.
//
// WHY query daily_stats instead of sessions? daily_stats is already
// aggregated — we don't care about individual sessions here, just
// whether ANY session happened on a given day.
func (m *Manager) calcStreak() int {
	rows, err := m.store.DB.Query(
		`SELECT date FROM daily_stats WHERE session_count > 0 ORDER BY date DESC LIMIT 365`,
	)
	if err != nil {
		return 0
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err == nil {
			dates = append(dates, d)
		}
	}

	if len(dates) == 0 {
		return 0
	}

	// Normalize "today" / "yesterday" for grace period —
	// a user who hasn't yet focused today still has their streak alive
	// until tomorrow. Missing two days in a row breaks it.
	today := time.Now()
	expect := today

	// If the most recent logged day is today, start from today.
	// If it's yesterday (user hasn't focused yet today), still count.
	// Otherwise, streak is 0.
	mostRecent, _ := time.Parse("2006-01-02", dates[0])
	daysSinceRecent := int(today.Sub(mostRecent).Hours() / 24)
	if daysSinceRecent > 1 {
		return 0
	}
	expect = mostRecent

	streak := 0
	for _, dstr := range dates {
		d, err := time.Parse("2006-01-02", dstr)
		if err != nil {
			break
		}
		if sameDay(d, expect) {
			streak++
			expect = expect.AddDate(0, 0, -1)
		} else {
			break
		}
	}
	return streak
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// GetRecentSessions returns the N most recent completed/interrupted sessions
// joined with their workspace name for display.
func (m *Manager) GetRecentSessions(limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := m.store.DB.Query(
		`SELECT
			s.id, s.workspace_id, COALESCE(w.name, '<deleted>'),
			s.task_description, COALESCE(s.commit_message, ''),
			COALESCE(s.duration_actual, 0), s.duration_planned,
			s.started_at, COALESCE(s.completed_at, ''),
			s.status, COALESCE(s.breach_attempts, 0)
		 FROM sessions s
		 LEFT JOIN workspaces w ON w.id = s.workspace_id
		 WHERE s.status != 'active'
		 ORDER BY s.started_at DESC
		 LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID, &s.WorkspaceID, &s.WorkspaceName,
			&s.TaskDescription, &s.CommitMessage,
			&s.DurationActual, &s.DurationPlanned,
			&s.StartedAt, &s.CompletedAt,
			&s.Status, &s.BreachAttempts,
		)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ExportData is the full JSON export payload. Users own their data —
// this lets them take it elsewhere.
type ExportData struct {
	ExportedAt string            `json:"exported_at"`
	Version    string            `json:"version"`
	Sessions   []Session         `json:"sessions"`
	DailyStats []DailyStatRecord `json:"daily_stats"`
}

// DailyStatRecord mirrors the daily_stats table for export.
type DailyStatRecord struct {
	Date                string `json:"date"`
	TotalFocusSeconds   int    `json:"total_focus_seconds"`
	SessionCount        int    `json:"session_count"`
	BlockedAttemptCount int    `json:"blocked_attempt_count"`
}

// ExportJSON returns a JSON-serializable dump of all session + daily_stats data.
func (m *Manager) ExportJSON() (string, error) {
	sessions, err := m.GetRecentSessions(100000) // effectively "all"
	if err != nil {
		return "", err
	}

	rows, err := m.store.DB.Query(
		`SELECT date, total_focus_seconds, session_count, blocked_attempt_count
		 FROM daily_stats ORDER BY date DESC`,
	)
	if err != nil {
		return "", fmt.Errorf("query daily_stats: %w", err)
	}
	defer rows.Close()

	var daily []DailyStatRecord
	for rows.Next() {
		var d DailyStatRecord
		if err := rows.Scan(&d.Date, &d.TotalFocusSeconds, &d.SessionCount, &d.BlockedAttemptCount); err != nil {
			return "", err
		}
		daily = append(daily, d)
	}

	payload := ExportData{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Version:    "0.1.0",
		Sessions:   sessions,
		DailyStats: daily,
	}

	buf, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(buf), nil
}
