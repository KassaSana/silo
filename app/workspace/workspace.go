package workspace

/*
 * workspace.go — CRUD operations for workspaces.
 *
 * CONCEPT: A Workspace is the core data model of silo. It defines
 * what you're ALLOWED to use during a focus session:
 *   - allowed_apps:  ["VS Code", "Terminal", "Chrome"]
 *   - allowed_sites: ["localhost:*", "github.com", "claude.ai"]
 *   - obsidian link: optional vault + note to auto-open
 *
 * These methods run SQL queries against the store and return Go structs.
 * Wails automatically serializes Go structs to JSON for the frontend.
 *
 * WHY return errors instead of panicking?
 * Go convention. The caller decides how to handle the error (show it
 * to the user, log it, retry, etc.). Panics crash the whole app.
 */

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"silo/app/store"
)

// Workspace is the data model the frontend sees.
// Wails converts this to a TypeScript interface automatically.
type Workspace struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	AllowedApps    []string `json:"allowed_apps"`
	AllowedSites   []string `json:"allowed_sites"`
	ObsidianVault  string   `json:"obsidian_vault"`
	ObsidianNote   string   `json:"obsidian_note"`
	TemplateSource string   `json:"template_source"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// Manager handles workspace operations. It holds a reference to the store.
type Manager struct {
	store *store.Store
}

// NewManager creates a workspace manager.
func NewManager(s *store.Store) *Manager {
	return &Manager{store: s}
}

// List returns all workspaces, ordered by creation date.
func (m *Manager) List() ([]Workspace, error) {
	rows, err := m.store.DB.Query(
		`SELECT id, name, allowed_apps, allowed_sites,
				obsidian_vault, obsidian_note, template_source,
				created_at, updated_at
		 FROM workspaces ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []Workspace
	for rows.Next() {
		ws, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, rows.Err()
}

// Get returns a single workspace by ID.
func (m *Manager) Get(id string) (Workspace, error) {
	row := m.store.DB.QueryRow(
		`SELECT id, name, allowed_apps, allowed_sites,
				obsidian_vault, obsidian_note, template_source,
				created_at, updated_at
		 FROM workspaces WHERE id = ?`, id,
	)
	return scanWorkspaceRow(row)
}

// Create inserts a new workspace and returns it.
func (m *Manager) Create(name string, apps, sites []string, obsVault, obsNote, templateSrc string) (Workspace, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	appsJSON, _ := json.Marshal(apps)
	sitesJSON, _ := json.Marshal(sites)

	_, err := m.store.DB.Exec(
		`INSERT INTO workspaces (id, name, allowed_apps, allowed_sites, obsidian_vault, obsidian_note, template_source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, string(appsJSON), string(sitesJSON), obsVault, obsNote, templateSrc, now, now,
	)
	if err != nil {
		return Workspace{}, fmt.Errorf("create workspace: %w", err)
	}

	return m.Get(id)
}

// Update modifies an existing workspace.
func (m *Manager) Update(id, name string, apps, sites []string, obsVault, obsNote string) (Workspace, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	appsJSON, _ := json.Marshal(apps)
	sitesJSON, _ := json.Marshal(sites)

	_, err := m.store.DB.Exec(
		`UPDATE workspaces SET name=?, allowed_apps=?, allowed_sites=?, obsidian_vault=?, obsidian_note=?, updated_at=?
		 WHERE id=?`,
		name, string(appsJSON), string(sitesJSON), obsVault, obsNote, now, id,
	)
	if err != nil {
		return Workspace{}, fmt.Errorf("update workspace: %w", err)
	}

	return m.Get(id)
}

// Delete removes a workspace by ID.
func (m *Manager) Delete(id string) error {
	_, err := m.store.DB.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	return nil
}

// ── Internal helpers ──

// scanWorkspace reads a workspace from a rows iterator.
func scanWorkspace(rows *sql.Rows) (Workspace, error) {
	var ws Workspace
	var appsJSON, sitesJSON string
	var obsVault, obsNote, tmplSrc sql.NullString

	err := rows.Scan(
		&ws.ID, &ws.Name, &appsJSON, &sitesJSON,
		&obsVault, &obsNote, &tmplSrc,
		&ws.CreatedAt, &ws.UpdatedAt,
	)
	if err != nil {
		return ws, fmt.Errorf("scan workspace: %w", err)
	}

	json.Unmarshal([]byte(appsJSON), &ws.AllowedApps)
	json.Unmarshal([]byte(sitesJSON), &ws.AllowedSites)
	ws.ObsidianVault = obsVault.String
	ws.ObsidianNote = obsNote.String
	ws.TemplateSource = tmplSrc.String

	return ws, nil
}

// scanWorkspaceRow reads from a single-row query.
func scanWorkspaceRow(row *sql.Row) (Workspace, error) {
	var ws Workspace
	var appsJSON, sitesJSON string
	var obsVault, obsNote, tmplSrc sql.NullString

	err := row.Scan(
		&ws.ID, &ws.Name, &appsJSON, &sitesJSON,
		&obsVault, &obsNote, &tmplSrc,
		&ws.CreatedAt, &ws.UpdatedAt,
	)
	if err != nil {
		return ws, fmt.Errorf("scan workspace: %w", err)
	}

	json.Unmarshal([]byte(appsJSON), &ws.AllowedApps)
	json.Unmarshal([]byte(sitesJSON), &ws.AllowedSites)
	ws.ObsidianVault = obsVault.String
	ws.ObsidianNote = obsNote.String
	ws.TemplateSource = tmplSrc.String

	return ws, nil
}
