package store

/*
 * db.go — SQLite connection and lifecycle management.
 *
 * CONCEPT: This is the "data access layer." It owns the database connection
 * and provides it to other packages (workspace, session, stats).
 *
 * WHY a Store struct instead of a global variable?
 * Testability. With a struct, you can create a test database in /tmp,
 * run your tests, and throw it away. A global makes that hard.
 *
 * The database file lives at ~/.silo/silo.db — following the convention
 * of putting app data in a dotfile directory in the user's home folder.
 */

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // Pure-Go SQLite driver — registers itself with database/sql
)

// Store holds the SQLite connection.
type Store struct {
	DB *sql.DB
}

// New opens (or creates) the SQLite database and runs migrations.
// Call Close() when done (typically on app shutdown).
func New() (*Store, error) {
	dbPath, err := dbFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// database/sql is Go's standard database interface.
	// "sqlite" is the driver name registered by modernc.org/sqlite.
	// The pragma settings:
	//   _journal_mode=WAL   — Write-Ahead Logging for better concurrent reads
	//   _busy_timeout=5000  — wait 5s if db is locked instead of failing immediately
	//   _foreign_keys=on    — enforce foreign key constraints
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Verify the connection actually works
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	store := &Store{DB: db}

	// Run schema migrations on every startup.
	// This is safe because CREATE TABLE IF NOT EXISTS is idempotent.
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}

// Close shuts down the database connection cleanly.
func (s *Store) Close() error {
	return s.DB.Close()
}

// dbFilePath returns ~/.silo/silo.db
func dbFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".silo", "silo.db"), nil
}
