package workspace

/*
 * obsidian.go — Obsidian vault integration.
 *
 * CONCEPT: Two tiny features, both intentionally dumb.
 *
 * 1. OpenNote() — on seal, launch `obsidian://open?vault=X&file=Y`
 *    via the OS URL handler. macOS: `open`. Windows: `start`.
 *    No Obsidian SDK. No plugin. Just the URI scheme.
 *
 * 2. AppendToDailyNote() — on session complete, append a markdown
 *    log entry to the user's daily note. The vault filesystem path
 *    and daily-note-path come from the settings table (user-configured).
 *    If vault_fs_path is empty, this is a no-op — obsidian integration
 *    opt-in rather than automatic.
 *
 * From the design spec: "Obsidian is just URI launch + file append.
 * Don't over-engineer."
 */

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SessionLogEntry is everything we need to format a markdown log line.
type SessionLogEntry struct {
	WorkspaceName   string
	TaskDescription string
	CommitMessage   string
	StartedAt       time.Time
	DurationActual  int // seconds
	BreachAttempts  int
	Exceptions      []string
}

// OpenNote launches the given vault/note via obsidian:// URI.
// Empty vault or note is a no-op (workspace didn't configure Obsidian).
func OpenNote(vault, note string) error {
	if vault == "" || note == "" {
		return nil
	}

	// obsidian://open?vault=MyVault&file=path/to/note
	// Note: Obsidian's URI handler accepts paths without .md extension.
	params := url.Values{}
	params.Set("vault", vault)
	params.Set("file", note)
	uri := "obsidian://open?" + params.Encode()

	return launchURI(uri)
}

// launchURI hands the URI off to the OS URL handler.
// macOS: `open <uri>`. Windows: `cmd /c start "" <uri>`.
func launchURI(uri string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", uri)
	case "windows":
		// The empty "" is cmd.exe's window title argument — required
		// when the first quoted token would otherwise be mistaken for it.
		cmd = exec.Command("cmd", "/c", "start", "", uri)
	default:
		// Linux fallback — xdg-open is standard but silo is macOS+Win only.
		cmd = exec.Command("xdg-open", uri)
	}
	return cmd.Start()
}

// AppendToDailyNote writes a session log entry to the day's daily note
// inside the vault. vaultFsPath is the absolute filesystem path to the
// vault (NOT the obsidian:// vault name). dailyNotePath is the relative
// subpath within the vault — e.g. "daily/2026-04-16.md" or just "daily".
// If dailyNotePath is a directory, we write to <dir>/<YYYY-MM-DD>.md.
// If it includes a template like "daily/{date}.md", we substitute {date}.
func AppendToDailyNote(vaultFsPath, dailyNotePath string, entry SessionLogEntry) error {
	if vaultFsPath == "" || dailyNotePath == "" {
		return nil // opt-in; silently skip if unconfigured
	}

	date := time.Now().Format("2006-01-02")
	resolved := strings.ReplaceAll(dailyNotePath, "{date}", date)
	resolved = strings.ReplaceAll(resolved, "{{date}}", date)

	// If the resolved path doesn't end in .md, treat it as a directory
	// and create today's file inside it.
	if !strings.HasSuffix(strings.ToLower(resolved), ".md") {
		resolved = filepath.Join(resolved, date+".md")
	}

	fullPath := filepath.Join(vaultFsPath, resolved)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("create daily note directory: %w", err)
	}

	// Build the markdown snippet
	md := formatSessionLog(entry)

	// Append (O_APPEND is atomic on POSIX for single writes of < PIPE_BUF).
	// We prepend a blank line to separate from whatever is already there.
	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open daily note: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + md); err != nil {
		return fmt.Errorf("write daily note: %w", err)
	}
	return nil
}

// formatSessionLog produces the markdown bullet block for a session.
// Example output:
//
//   ## 🔒 silo — 10:30 (90m, react-project)
//   - **task:** build auth flow
//   - **commit:** landed OAuth2 callback
//   - **exceptions:** site:docs.python.org
func formatSessionLog(e SessionLogEntry) string {
	var b strings.Builder

	timeStr := e.StartedAt.Local().Format("15:04")
	mins := e.DurationActual / 60
	fmt.Fprintf(&b, "## silo — %s (%dm, %s)\n", timeStr, mins, e.WorkspaceName)

	if e.TaskDescription != "" {
		fmt.Fprintf(&b, "- **task:** %s\n", e.TaskDescription)
	}
	if e.CommitMessage != "" {
		fmt.Fprintf(&b, "- **commit:** %s\n", e.CommitMessage)
	}
	if e.BreachAttempts > 0 {
		fmt.Fprintf(&b, "- **breach attempts:** %d\n", e.BreachAttempts)
	}
	if len(e.Exceptions) > 0 {
		fmt.Fprintf(&b, "- **exceptions:** %s\n", strings.Join(e.Exceptions, ", "))
	}

	return b.String()
}
