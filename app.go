package main

import (
	"context"
	"fmt"

	"silo/app/blocker"
	"silo/app/session"
	"silo/app/stats"
	"silo/app/store"
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
	return a.workspace.Create(name, apps, sites, obsVault, obsNote, templateSrc)
}

func (a *App) UpdateWorkspace(id, name string, apps, sites []string, obsVault, obsNote string) (workspace.Workspace, error) {
	if a.workspace == nil {
		return workspace.Workspace{}, fmt.Errorf("not initialized")
	}
	return a.workspace.Update(id, name, apps, sites, obsVault, obsNote)
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
	for _, t := range workspace.BuiltinTemplates() {
		if t.Name == templateName {
			return a.workspace.Create(workspaceName, t.Apps, t.Sites, "", "", templateName)
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

	// Get workspace to know what's allowed
	ws, err := a.workspace.Get(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}

	return a.session.Seal(
		workspaceID, ws.Name, task, firstStep,
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
	return a.session.AddException(exceptionType, value)
}

// CompleteSession finishes the session with a commit message.
func (a *App) CompleteSession(commitMessage string) error {
	if a.session == nil {
		return fmt.Errorf("not initialized")
	}
	return a.session.Complete(commitMessage)
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
