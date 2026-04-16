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

	// Ensure the directory exists. 0700 because the SQLite file lives here
	// and may contain user task descriptions, timestamps, etc. — user-only
	// access matches the file permission we enforce below.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
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

	// Enforce user-only permissions on the db + WAL sidecars.
	// CLAUDE.md requires 0600. We chmod AFTER Ping so the files definitely
	// exist — SQLite creates them lazily. A looser umask on the host would
	// otherwise leave silo.db world-readable on first run.
	// WHY also chmod the parent dir: a prior run may have created ~/.silo
	// with 0755 before this hardening landed; MkdirAll is a no-op in that case.
	if err := os.Chmod(filepath.Dir(dbPath), 0700); err != nil {
		// Non-fatal: db itself is still chmod'd below. Log and continue.
		fmt.Printf("warn: chmod db dir: %v\n", err)
	}
	tightenDBPerms(dbPath)

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

// tightenDBPerms chmods silo.db and its WAL sidecars (-wal, -shm) to 0600.
//
// WHY all three: WAL mode splits the database into three files. The -wal and
// -shm contain page images for in-flight transactions — functionally part of
// the database, and equally sensitive. Leaving them 0644 defeats the purpose
// of locking down silo.db itself.
//
// Best-effort: chmod failures are logged but don't abort app startup. A user
// on Windows (where Unix permissions don't apply) gets a harmless error.
func tightenDBPerms(dbPath string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		p := dbPath + suffix
		// Skip sidecars that don't exist yet (SQLite creates them lazily
		// on first write; a brand-new db won't have them).
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := os.Chmod(p, 0600); err != nil {
			fmt.Printf("warn: chmod %s: %v\n", p, err)
		}
	}
}
