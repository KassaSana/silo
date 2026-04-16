package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// AppendToDailyNote does filesystem I/O, but it's enclosed inside a
// user-supplied vault path. Using t.TempDir() keeps the test hermetic.

func TestAppendToDailyNote_UnconfiguredIsNoOp(t *testing.T) {
	// Obsidian integration is opt-in. Empty vault or note path must not
	// error, otherwise every non-Obsidian user sees a failure on every
	// session complete.
	if err := AppendToDailyNote("", "daily/2026-04-16.md", SessionLogEntry{}); err != nil {
		t.Errorf("empty vault should be no-op, got %v", err)
	}
	if err := AppendToDailyNote("/tmp", "", SessionLogEntry{}); err != nil {
		t.Errorf("empty note should be no-op, got %v", err)
	}
}

func TestAppendToDailyNote_DateSubstitution(t *testing.T) {
	// {date} / {{date}} get replaced with today's YYYY-MM-DD.
	vault := t.TempDir()
	today := time.Now().Format("2006-01-02")

	entry := SessionLogEntry{
		WorkspaceName:   "coding",
		TaskDescription: "test",
		StartedAt:       time.Date(2026, 4, 16, 10, 30, 0, 0, time.Local),
		DurationActual:  60 * 60,
	}

	if err := AppendToDailyNote(vault, "daily/{date}.md", entry); err != nil {
		t.Fatalf("AppendToDailyNote: %v", err)
	}

	expected := filepath.Join(vault, "daily", today+".md")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file %s missing: %v", expected, err)
	}
}

func TestAppendToDailyNote_DirectoryPath(t *testing.T) {
	// If dailyNotePath is a directory (no .md), we drop today's file inside it.
	vault := t.TempDir()
	today := time.Now().Format("2006-01-02")

	err := AppendToDailyNote(vault, "daily", SessionLogEntry{
		WorkspaceName: "x",
		StartedAt:     time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(vault, "daily", today+".md")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected %s, got error: %v", expected, err)
	}
}

func TestAppendToDailyNote_Appends(t *testing.T) {
	// Second call appends — must not overwrite. Critical: multiple sessions
	// in one day should all be recorded.
	vault := t.TempDir()
	note := "today.md"

	for i := 0; i < 3; i++ {
		err := AppendToDailyNote(vault, note, SessionLogEntry{
			WorkspaceName: "w",
			CommitMessage: "run",
			StartedAt:     time.Now(),
		})
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(vault, note))
	if err != nil {
		t.Fatal(err)
	}
	// Three "## silo —" headings should be present.
	if strings.Count(string(data), "## silo —") != 3 {
		t.Errorf("expected 3 headings, got content:\n%s", data)
	}
}

func TestAppendToDailyNote_CreatesParentDirs(t *testing.T) {
	// A path of "journals/daily/today.md" must auto-create both dirs.
	vault := t.TempDir()

	err := AppendToDailyNote(vault, "journals/daily/today.md", SessionLogEntry{
		WorkspaceName: "w",
		StartedAt:     time.Now(),
	})
	if err != nil {
		t.Fatalf("AppendToDailyNote: %v", err)
	}

	if _, err := os.Stat(filepath.Join(vault, "journals", "daily", "today.md")); err != nil {
		t.Errorf("nested file not created: %v", err)
	}
}

func TestAppendToDailyNote_IncludesTaskAndCommit(t *testing.T) {
	// The markdown must contain the task description + commit message.
	vault := t.TempDir()

	err := AppendToDailyNote(vault, "note.md", SessionLogEntry{
		WorkspaceName:   "coding",
		TaskDescription: "build OAuth flow",
		CommitMessage:   "landed callback handler",
		StartedAt:       time.Now(),
		DurationActual:  90 * 60,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(vault, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{"build OAuth flow", "landed callback handler", "coding"} {
		if !strings.Contains(s, want) {
			t.Errorf("note missing %q:\n%s", want, s)
		}
	}
}
