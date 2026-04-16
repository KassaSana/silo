package store

/*
 * migrations.go — Database schema creation.
 *
 * CONCEPT: Migrations define the shape of your data. This runs on every
 * app startup, but "IF NOT EXISTS" makes it safe to run repeatedly.
 *
 * The schema matches the design spec exactly:
 *   - workspaces: named configs with allowed apps/sites + Obsidian link
 *   - sessions: focus session records with task, timer, lock, exceptions
 *   - daily_stats: aggregated daily focus metrics for fast queries
 *   - settings: key-value pairs for app configuration
 *
 * WHY store allowed_apps/sites as JSON text instead of separate tables?
 * Simplicity. These are small lists (3-10 items) that are always read
 * and written together. A join table would add complexity for no benefit.
 * This is a pragmatic choice — not everything needs full normalization.
 */

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			allowed_apps TEXT NOT NULL DEFAULT '[]',
			allowed_sites TEXT NOT NULL DEFAULT '[]',
			obsidian_vault TEXT,
			obsidian_note TEXT,
			template_source TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			task_description TEXT NOT NULL,
			first_step TEXT NOT NULL,
			commit_message TEXT,
			lock_type TEXT NOT NULL,
			lock_chars INTEGER,
			duration_planned INTEGER NOT NULL,
			duration_actual INTEGER,
			started_at DATETIME NOT NULL,
			completed_at DATETIME,
			status TEXT NOT NULL DEFAULT 'active',
			breach_attempts INTEGER DEFAULT 0,
			blocked_attempts TEXT DEFAULT '[]',
			quick_exceptions TEXT DEFAULT '[]',
			FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
		)`,

		`CREATE TABLE IF NOT EXISTS daily_stats (
			date TEXT PRIMARY KEY,
			total_focus_seconds INTEGER DEFAULT 0,
			session_count INTEGER DEFAULT 0,
			blocked_attempt_count INTEGER DEFAULT 0
		)`,

		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	// Default settings — INSERT OR IGNORE means "only insert if key doesn't exist"
	// This seeds the defaults on first run without overwriting user changes.
	defaults := []string{
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('default_duration', '5400')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('default_lock_type', 'random-text')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('default_lock_chars', '200')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('escalation_step', '200')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('dnd_integration', 'true')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('show_blocked_log', 'true')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('block_page_port', '9512')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('obsidian_daily_note_path', '')`,
		`INSERT OR IGNORE INTO settings (key, value) VALUES ('obsidian_vault_fs_path', '')`,
	}

	for _, m := range migrations {
		if _, err := s.DB.Exec(m); err != nil {
			return err
		}
	}

	for _, d := range defaults {
		if _, err := s.DB.Exec(d); err != nil {
			return err
		}
	}

	return nil
}
