//go:build darwin

package platform

/*
 * dnd_darwin.go — macOS Focus / Do Not Disturb toggle.
 *
 * APPROACH: macOS 12+ replaced the old "Do Not Disturb" plist toggle
 * with Focus modes, which are intentionally hard to flip programmatically.
 * Best practical path: Shortcuts.app. Users (or silo on first run) create
 * a one-line shortcut named "silo-dnd-on" / "silo-dnd-off" that sets a
 * Focus filter, and we invoke it via `shortcuts run`.
 *
 * If the shortcut doesn't exist, this is a silent no-op — we log and move
 * on. Users who don't want DND integration just never create the shortcut.
 *
 * From design spec: "Auto-enable DND (macOS Focus / Windows Focus Assist)
 * on seal." Best-effort is the right bar here.
 */

import (
	"fmt"
	"os/exec"
)

const (
	shortcutOn  = "silo-dnd-on"
	shortcutOff = "silo-dnd-off"
)

// EnableDND turns on Focus mode via the user's "silo-dnd-on" shortcut.
// No-op if the shortcut isn't installed (Shortcuts.app returns non-zero).
func EnableDND() error {
	if err := runShortcut(shortcutOn); err != nil {
		return fmt.Errorf("enable dnd: %w", err)
	}
	return nil
}

// DisableDND turns off Focus mode via "silo-dnd-off".
func DisableDND() error {
	if err := runShortcut(shortcutOff); err != nil {
		return fmt.Errorf("disable dnd: %w", err)
	}
	return nil
}

// runShortcut invokes the `shortcuts` CLI. Present on macOS 12+.
func runShortcut(name string) error {
	cmd := exec.Command("shortcuts", "run", name)
	return cmd.Run()
}
