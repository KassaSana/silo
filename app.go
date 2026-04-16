package main

import (
	"context"
	"fmt"

	"silo/app/blocker"
	"silo/app/session"
	"silo/app/stats"
	"silo/app/store"
	"silo/app/validate"
	"silo/app/workspace"
)

type App struct {
	ctx       context.Context
	store     *store.Store
	workspace *workspace.Manager
	session   *session.Manager
	stats     *stats.Manager
	engine    *blocker.Engine
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	s, err := store.New()
	if err != nil {
		fmt.Printf("FATAL: database init failed: %v\n", err)
		return
	}
	a.store = s
	a.workspace = workspace.NewManager(s)
	a.engine = blocker.NewEngine()
	a.session = session.NewManager(ctx, s, a.engine)
	a.stats = stats.NewManager(s)

	// Crash recovery — clean up any sessions left 'active' from a prior run.
	if err := a.session.RecoverInterrupted(); err != nil {
		fmt.Printf("warn: crash recovery: %v\n", err)
	}
}

// ── Workspace bindings ──

func (a *App) ListWorkspaces() ([]workspace.Workspace, error) {
	if a.workspace == nil {
		return nil, fmt.Errorf("not initialized")
	}
	return a.workspace.List()
}

func (a *App) GetWorkspace(id string) (workspace.Workspace, error) {
	if a.workspace == nil {
		return workspace.Workspace{}, fmt.Errorf("not initialized")
	}
	return a.workspace.Get(id)
}

func (a *App) CreateWorkspace(name string, apps, sites []string, obsVault, obsNote, templateSrc string) (workspace.Workspace, error) {
	if a.workspace == nil {
		return workspace.Workspace{}, fmt.Errorf("not initialized")
	}
	cleanName, cleanApps, cleanSites, cleanVault, cleanNote, err := validateWorkspaceInputs(name, apps, sites, obsVault, obsNote)
	if err != nil {
		return workspace.Workspace{}, err
	}
	return a.workspace.Create(cleanName, cleanApps, cleanSites, cleanVault, cleanNote, templateSrc)
}

func (a *App) UpdateWorkspace(id, name string, apps, sites []string, obsVault, obsNote string) (workspace.Workspace, error) {
	if a.workspace == nil {
		return workspace.Workspace{}, fmt.Errorf("not initialized")
	}
	cleanName, cleanApps, cleanSites, cleanVault, cleanNote, err := validateWorkspaceInputs(name, apps, sites, obsVault, obsNote)
	if err != nil {
		return workspace.Workspace{}, err
	}
	return a.workspace.Update(id, cleanName, cleanApps, cleanSites, cleanVault, cleanNote)
}

// validateWorkspaceInputs is the shared boundary check for Create + Update.
// All fields are normalised (control chars stripped, whitespace trimmed) and
// rejected if invalid. Domain validation is strict because these values end up
// concatenated into /etc/hosts entries downstream.
func validateWorkspaceInputs(name string, apps, sites []string, obsVault, obsNote string) (
	string, []string, []string, string, string, error,
) {
	cleanName, err := validate.Text(name, validate.MaxTextLen)
	if err != nil {
		return "", nil, nil, "", "", fmt.Errorf("workspace name: %w", err)
	}
	cleanApps := make([]string, 0, len(apps))
	for i, a := range apps {
		v, err := validate.AppName(a)
		if err != nil {
			return "", nil, nil, "", "", fmt.Errorf("app #%d (%q): %w", i+1, a, err)
		}
		cleanApps = append(cleanApps, v)
	}
	cleanSites := make([]string, 0, len(sites))
	for i, s := range sites {
		// Trim whitespace at the boundary — common when users paste — but
		// Domain() itself rejects any in-string whitespace.
		trimmed := trimSpaceStrict(s)
		if err := validate.Domain(trimmed); err != nil {
			return "", nil, nil, "", "", fmt.Errorf("site #%d (%q): %w", i+1, s, err)
		}
		cleanSites = append(cleanSites, trimmed)
	}
	cleanVault, err := validate.Path(obsVault)
	if err != nil {
		return "", nil, nil, "", "", fmt.Errorf("obsidian vault: %w", err)
	}
	cleanNote, err := validate.Path(obsNote)
	if err != nil {
		return "", nil, nil, "", "", fmt.Errorf("obsidian note: %w", err)
	}
	return cleanName, cleanApps, cleanSites, cleanVault, cleanNote, nil
}

// trimSpaceStrict trims ASCII whitespace only (space, tab, CR, LF). We don't
// use strings.TrimSpace because it's Unicode-aware and we want byte-exact
// behaviour for domains.
func trimSpaceStrict(s string) string {
	start := 0
	for start < len(s) {
		c := s[start]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			start++
			continue
		}
		break
	}
	end := len(s)
	for end > start {
		c := s[end-1]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			end--
			continue
		}
		break
	}
	return s[start:end]
}

func (a *App) DeleteWorkspace(id string) error {
	if a.workspace == nil {
		return fmt.Errorf("not initialized")
	}
	return a.workspace.Delete(id)
}

func (a *App) GetTemplates() []workspace.Template {
	return workspace.BuiltinTemplates()
}

func (a *App) CreateFromTemplate(templateName, workspaceName string) (workspace.Workspace, error) {
	if a.workspace == nil {
		return workspace.Workspace{}, fmt.Errorf("not initialized")
	}
	// Only the user-supplied workspace name needs validation here; template
	// apps/sites come from BuiltinTemplates() (trusted constants).
	cleanName, err := validate.Text(workspaceName, validate.MaxTextLen)
	if err != nil {
		return workspace.Workspace{}, fmt.Errorf("workspace name: %w", err)
	}
	for _, t := range workspace.BuiltinTemplates() {
		if t.Name == templateName {
			return a.workspace.Create(cleanName, t.Apps, t.Sites, "", "", templateName)
		}
	}
	return workspace.Workspace{}, fmt.Errorf("template %q not found", templateName)
}

// ── Session bindings ──

// SealWorkspace starts a focus session. This is the big one.
func (a *App) SealWorkspace(workspaceID, task, firstStep string,
	lockType string, lockChars, durationMinutes int) (*session.Session, error) {

	if a.session == nil {
		return nil, fmt.Errorf("not initialized")
	}

	// Task + first step flow into SQLite, into the Obsidian daily note markdown,
	// and into the block page UI. Strip control chars at the boundary so a
	// pasted newline doesn't break the markdown format downstream.
	cleanTask, err := validate.Text(task, validate.MaxTextLen)
	if err != nil {
		return nil, fmt.Errorf("task description: %w", err)
	}
	cleanFirstStep, err := validate.Text(firstStep, validate.MaxTextLen)
	if err != nil {
		return nil, fmt.Errorf("first step: %w", err)
	}
	if durationMinutes <= 0 || durationMinutes > 24*60 {
		return nil, fmt.Errorf("duration out of range (1-1440 minutes)")
	}
	if lockChars < 0 || lockChars > 10_000 {
		return nil, fmt.Errorf("lock length out of range")
	}

	// Get workspace to know what's allowed
	ws, err := a.workspace.Get(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}

	return a.session.Seal(
		workspaceID, ws.Name, cleanTask, cleanFirstStep,
		session.LockType(lockType), lockChars, durationMinutes,
		ws.AllowedApps, ws.AllowedSites,
		ws.ObsidianVault, ws.ObsidianNote,
	)
}

// GetCurrentSession returns the active session (null if none).
func (a *App) GetCurrentSession() *session.Session {
	if a.session == nil {
		return nil
	}
	return a.session.GetCurrent()
}

// AttemptUnlock tries to break the seal.
func (a *App) AttemptUnlock(input string) (bool, string, int, error) {
	if a.session == nil {
		return false, "", 0, fmt.Errorf("not initialized")
	}
	return a.session.AttemptUnlock(input)
}

// AddException adds a quick exception during active seal.
func (a *App) AddException(exceptionType, value, confirmation string) error {
	if confirmation != "i need this" {
		return fmt.Errorf("type 'i need this' to confirm")
	}
	if a.session == nil {
		return fmt.Errorf("not initialized")
	}
	// Exception values flow straight into the live blocker (hosts file or
	// process allowlist). Validate by type — a "site" exception is a domain,
	// an "app" exception is a process name.
	var cleanValue string
	var err error
	switch exceptionType {
	case "site":
		cleanValue = trimSpaceStrict(value)
		if err = validate.Domain(cleanValue); err != nil {
			return fmt.Errorf("exception site: %w", err)
		}
	case "app":
		cleanValue, err = validate.AppName(value)
		if err != nil {
			return fmt.Errorf("exception app: %w", err)
		}
	default:
		return fmt.Errorf("unknown exception type %q (expected site or app)", exceptionType)
	}
	return a.session.AddException(exceptionType, cleanValue)
}

// CompleteSession finishes the session with a commit message.
func (a *App) CompleteSession(commitMessage string) error {
	if a.session == nil {
		return fmt.Errorf("not initialized")
	}
	// Commit message is optional (empty = "no message"), but if provided
	// it must be sanitised because it's written into the Obsidian daily note.
	cleanMsg := commitMessage
	if commitMessage != "" {
		c, err := validate.Text(commitMessage, validate.MaxTextLen)
		if err != nil {
			return fmt.Errorf("commit message: %w", err)
		}
		cleanMsg = c
	}
	return a.session.Complete(cleanMsg)
}

// GetBlockedAttempts returns recently blocked processes for the UI.
func (a *App) GetBlockedAttempts() []blocker.BlockedAttempt {
	if a.session == nil {
		return nil
	}
	return a.session.GetBlockedAttempts()
}

// ── Stats bindings ──

// GetStatsSummary returns today/week/streak/lifetime totals.
func (a *App) GetStatsSummary() (stats.Summary, error) {
	if a.stats == nil {
		return stats.Summary{}, fmt.Errorf("not initialized")
	}
	return a.stats.GetSummary()
}

// GetRecentSessions returns the N most recent sessions with workspace name joined.
func (a *App) GetRecentSessions(limit int) ([]stats.Session, error) {
	if a.stats == nil {
		return nil, fmt.Errorf("not initialized")
	}
	return a.stats.GetRecentSessions(limit)
}

// ExportStatsJSON returns a JSON blob of all session + daily_stats history.
// Frontend can trigger download or write to disk via save dialog.
func (a *App) ExportStatsJSON() (string, error) {
	if a.stats == nil {
		return "", fmt.Errorf("not initialized")
	}
	return a.stats.ExportJSON()
}
