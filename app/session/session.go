package session

/*
 * session.go — Session lifecycle management.
 *
 * LIFECYCLE: IDLE → CONFIGURING → SEALED → COMPLETED
 *
 * A session represents one focus block. It records:
 * - What workspace, what task, what lock type
 * - Duration planned vs actual
 * - All blocked attempts and quick exceptions
 * - Breach attempt count
 * - Commit message on completion
 *
 * The session manager coordinates:
 * - Creating the session record in SQLite
 * - Starting the blocking engine
 * - Running the timer
 * - Handling unlock attempts (with escalation)
 * - Handling quick exceptions
 * - Completing the session and recording stats
 *
 * CRASH RECOVERY: The session is written to SQLite on seal with
 * status='active'. On app launch, if an active session exists,
 * we know the app crashed mid-seal and can recover.
 */

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"silo/app/blocker"
	"silo/app/platform"
	"silo/app/store"
	"silo/app/workspace"
)

// State represents the current session state.
type State string

const (
	StateIdle        State = "idle"
	StateConfiguring State = "configuring"
	StateSealed      State = "sealed"
	StateCompleted   State = "completed"
)

// Session is the active session data.
type Session struct {
	ID              string   `json:"id"`
	WorkspaceID     string   `json:"workspace_id"`
	WorkspaceName   string   `json:"workspace_name"`
	TaskDescription string   `json:"task_description"`
	FirstStep       string   `json:"first_step"`
	CommitMessage   string   `json:"commit_message"`
	LockType        LockType `json:"lock_type"`
	LockChars       int      `json:"lock_chars"`
	LockText        string   `json:"lock_text"`
	DurationPlanned int      `json:"duration_planned"` // seconds
	StartedAt       string   `json:"started_at"`
	Status          State    `json:"status"`
	BreachAttempts  int      `json:"breach_attempts"`
	Exceptions      []string `json:"exceptions"`
	ObsidianVault   string   `json:"obsidian_vault"`
	ObsidianNote    string   `json:"obsidian_note"`
}

// Manager handles the session lifecycle.
type Manager struct {
	ctx     context.Context
	store   *store.Store
	engine  *blocker.Engine
	timer   *Timer
	current *Session
}

// NewManager creates a session manager.
func NewManager(ctx context.Context, s *store.Store, engine *blocker.Engine) *Manager {
	return &Manager{
		ctx:    ctx,
		store:  s,
		engine: engine,
	}
}

// Seal starts a new focus session.
func (m *Manager) Seal(workspaceID, workspaceName, task, firstStep string,
	lockType LockType, lockChars, durationMinutes int,
	allowedApps, allowedSites []string,
	obsidianVault, obsidianNote string) (*Session, error) {

	if m.current != nil {
		return nil, fmt.Errorf("session already active")
	}

	// Generate lock text if using random-text lock
	var lockText string
	if lockType == LockRandomText {
		var err error
		lockText, err = GenerateLockText(lockChars)
		if err != nil {
			return nil, fmt.Errorf("generate lock text: %w", err)
		}
	}

	durationSeconds := durationMinutes * 60
	now := time.Now().UTC().Format(time.RFC3339)

	session := &Session{
		ID:              uuid.New().String(),
		WorkspaceID:     workspaceID,
		WorkspaceName:   workspaceName,
		TaskDescription: task,
		FirstStep:       firstStep,
		LockType:        lockType,
		LockChars:       lockChars,
		LockText:        lockText,
		DurationPlanned: durationSeconds,
		StartedAt:       now,
		Status:          StateSealed,
		ObsidianVault:   obsidianVault,
		ObsidianNote:    obsidianNote,
	}

	// Write to SQLite immediately (crash recovery)
	if err := m.saveSession(session); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	// Engage blocking engine
	if err := m.engine.Engage(allowedApps, allowedSites, workspaceName, task); err != nil {
		return nil, fmt.Errorf("engage blocking: %w", err)
	}

	// Start countdown timer
	m.timer = NewTimer(m.ctx, durationSeconds)
	m.timer.Start()

	m.current = session

	// Obsidian auto-open. Non-fatal if it fails — the seal is already in
	// effect, and the user can open the note manually.
	if obsidianVault != "" && obsidianNote != "" {
		if err := workspace.OpenNote(obsidianVault, obsidianNote); err != nil {
			fmt.Printf("warn: obsidian open: %v\n", err)
		}
	}

	// DND/Focus Assist. Best-effort; failure doesn't abort the seal.
	if m.getSetting("dnd_integration") == "true" {
		if err := platform.EnableDND(); err != nil {
			fmt.Printf("warn: enable dnd: %v\n", err)
		}
	}

	return session, nil
}

// AttemptUnlock tries to break the seal. Returns (success, newLockText, newLockChars).
func (m *Manager) AttemptUnlock(input string) (bool, string, int, error) {
	if m.current == nil {
		return false, "", 0, fmt.Errorf("no active session")
	}

	if m.current.LockType == LockTimer {
		return false, "", 0, fmt.Errorf("timer lock cannot be unlocked early")
	}

	if m.current.LockType == LockReboot {
		return false, "", 0, fmt.Errorf("reboot lock requires restart")
	}

	// Validate input against lock text
	matches, _ := ValidateLockText(input, m.current.LockText)
	if matches {
		// Success — complete the session
		m.completeSession("unlocked (breach)")
		return true, "", 0, nil
	}

	// Failed attempt — escalate
	m.current.BreachAttempts++
	newChars := EscalateChars(m.current.BreachAttempts+1, 200, 200)

	// Generate new (longer) lock text
	newText, err := GenerateLockText(newChars)
	if err != nil {
		return false, "", 0, err
	}
	m.current.LockText = newText
	m.current.LockChars = newChars

	// Persist updated session
	m.saveSession(m.current)

	return false, newText, newChars, nil
}

// AddException adds a quick exception during the active seal.
func (m *Manager) AddException(exceptionType, value string) error {
	if m.current == nil {
		return fmt.Errorf("no active session")
	}

	if exceptionType == "site" {
		if err := m.engine.AddSiteException(value); err != nil {
			return err
		}
	} else if exceptionType == "app" {
		m.engine.AddAppException(value)
	}

	m.current.Exceptions = append(m.current.Exceptions, fmt.Sprintf("%s:%s", exceptionType, value))
	m.saveSession(m.current)
	return nil
}

// Complete finishes the session normally (timer expired).
func (m *Manager) Complete(commitMessage string) error {
	if m.current == nil {
		return fmt.Errorf("no active session")
	}
	m.current.CommitMessage = commitMessage
	m.completeSession("completed")
	return nil
}

// GetCurrent returns the active session (nil if none).
func (m *Manager) GetCurrent() *Session {
	return m.current
}

// RecoverInterrupted is called on app startup. If a prior run crashed
// while a session was active, the session row will still say status='active'.
// We mark those as 'interrupted', best-effort fill in duration_actual from
// started_at → now, update daily_stats, and clean up any orphaned hosts
// entries left behind by the crashed seal.
//
// WHY this matters: without recovery, next seal would fail ("session already
// active" in SQLite-inspected state) and the old hosts block would linger.
func (m *Manager) RecoverInterrupted() error {
	rows, err := m.store.DB.Query(
		`SELECT id, started_at FROM sessions WHERE status = 'active'`,
	)
	if err != nil {
		return fmt.Errorf("query active sessions: %w", err)
	}

	type active struct {
		id        string
		startedAt string
	}
	var actives []active
	for rows.Next() {
		var a active
		if err := rows.Scan(&a.id, &a.startedAt); err != nil {
			rows.Close()
			return fmt.Errorf("scan active: %w", err)
		}
		actives = append(actives, a)
	}
	rows.Close()

	now := time.Now().UTC()
	for _, a := range actives {
		// Compute how long the session was running before crash.
		var elapsed int
		if started, parseErr := time.Parse(time.RFC3339, a.startedAt); parseErr == nil {
			elapsed = int(now.Sub(started).Seconds())
			if elapsed < 0 {
				elapsed = 0
			}
		}

		_, _ = m.store.DB.Exec(
			`UPDATE sessions SET status='interrupted', duration_actual=?, completed_at=?
			 WHERE id=?`,
			elapsed, now.Format(time.RFC3339), a.id,
		)

		// Credit the partial session to daily_stats so the streak is honest.
		day := now.Format("2006-01-02")
		_, _ = m.store.DB.Exec(
			`INSERT INTO daily_stats (date, total_focus_seconds, session_count)
			 VALUES (?, ?, 1)
			 ON CONFLICT(date) DO UPDATE SET
			   total_focus_seconds = total_focus_seconds + ?,
			   session_count = session_count + 1`,
			day, elapsed, elapsed,
		)
	}

	// Even with zero interrupted rows, a prior ungraceful shutdown may have
	// left the hosts file modified. Strip silo markers defensively.
	if len(actives) > 0 {
		if err := blocker.CleanupOrphanedHosts(); err != nil {
			fmt.Printf("warn: hosts cleanup: %v\n", err)
		}
	}

	return nil
}

// GetBlockedAttempts returns recently blocked processes.
func (m *Manager) GetBlockedAttempts() []blocker.BlockedAttempt {
	return m.engine.GetBlockedAttempts()
}

// completeSession stops everything and records the session.
func (m *Manager) completeSession(status string) {
	if m.timer != nil {
		m.timer.Stop()
	}
	m.engine.Disengage()

	m.current.Status = State(status)

	// Update session in database
	elapsed := 0
	if m.timer != nil {
		elapsed = m.timer.Elapsed()
	}

	exceptionsJSON, _ := json.Marshal(m.current.Exceptions)
	blockedJSON, _ := json.Marshal(m.engine.GetBlockedAttempts())

	m.store.DB.Exec(
		`UPDATE sessions SET
			status=?, commit_message=?, duration_actual=?,
			breach_attempts=?, quick_exceptions=?, blocked_attempts=?,
			completed_at=?
		 WHERE id=?`,
		status, m.current.CommitMessage, elapsed,
		m.current.BreachAttempts, string(exceptionsJSON), string(blockedJSON),
		time.Now().UTC().Format(time.RFC3339), m.current.ID,
	)

	// Update daily stats
	today := time.Now().Format("2006-01-02")
	m.store.DB.Exec(
		`INSERT INTO daily_stats (date, total_focus_seconds, session_count)
		 VALUES (?, ?, 1)
		 ON CONFLICT(date) DO UPDATE SET
		   total_focus_seconds = total_focus_seconds + ?,
		   session_count = session_count + 1`,
		today, elapsed, elapsed,
	)

	// Append to Obsidian daily note if configured.
	// Non-fatal — missing vault config, misconfigured paths, etc. shouldn't
	// abort session cleanup.
	m.maybeAppendDailyNote(elapsed)

	// Restore notifications.
	if m.getSetting("dnd_integration") == "true" {
		if err := platform.DisableDND(); err != nil {
			fmt.Printf("warn: disable dnd: %v\n", err)
		}
	}

	m.current = nil
}

// maybeAppendDailyNote writes a session-log markdown block to the user's
// Obsidian daily note, if they've set both obsidian_vault_fs_path and
// obsidian_daily_note_path in settings.
func (m *Manager) maybeAppendDailyNote(elapsed int) {
	vaultFsPath := m.getSetting("obsidian_vault_fs_path")
	dailyNotePath := m.getSetting("obsidian_daily_note_path")

	if vaultFsPath == "" || dailyNotePath == "" {
		return
	}

	startedAt, _ := time.Parse(time.RFC3339, m.current.StartedAt)
	entry := workspace.SessionLogEntry{
		WorkspaceName:   m.current.WorkspaceName,
		TaskDescription: m.current.TaskDescription,
		CommitMessage:   m.current.CommitMessage,
		StartedAt:       startedAt,
		DurationActual:  elapsed,
		BreachAttempts:  m.current.BreachAttempts,
		Exceptions:      m.current.Exceptions,
	}

	if err := workspace.AppendToDailyNote(vaultFsPath, dailyNotePath, entry); err != nil {
		fmt.Printf("warn: obsidian daily note append: %v\n", err)
	}
}

// getSetting fetches a single value from the settings KV table.
// Returns "" if the row is missing.
func (m *Manager) getSetting(key string) string {
	var v string
	_ = m.store.DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	return v
}

// saveSession writes/updates the session in SQLite.
func (m *Manager) saveSession(s *Session) error {
	exceptionsJSON, _ := json.Marshal(s.Exceptions)

	_, err := m.store.DB.Exec(
		`INSERT OR REPLACE INTO sessions
		 (id, workspace_id, task_description, first_step, lock_type, lock_chars,
		  duration_planned, started_at, status, breach_attempts, quick_exceptions)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.WorkspaceID, s.TaskDescription, s.FirstStep,
		string(s.LockType), s.LockChars, s.DurationPlanned,
		s.StartedAt, string(s.Status), s.BreachAttempts, string(exceptionsJSON),
	)
	return err
}
