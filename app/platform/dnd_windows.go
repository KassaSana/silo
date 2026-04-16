//go:build windows

package platform

/*
 * dnd_windows.go — Windows Focus Assist toggle.
 *
 * APPROACH: Windows stores Focus Assist state in the registry under
 * HKCU\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\
 *   Windows.SystemToast.FocusAssist\Enabled (DWORD: 0/1)
 *
 * Writing the registry key alone doesn't always propagate — Windows
 * reads the "focus session" state from a background service. The reliable
 * path for toggling is the `focusassist.exe` utility or PowerShell's
 * `Set-WinUIBindings`, but neither ships on all versions.
 *
 * For MVP we use `reg add` to set the registry value; it works on Win11
 * and triggers the shell to honor it on next notification attempt. Best-
 * effort, same as macOS — log and continue on failure.
 */

import (
	"fmt"
	"os/exec"
)

const focusAssistKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\Windows.SystemToast.FocusAssist`

// EnableDND turns on Focus Assist (registry DWORD = 1).
func EnableDND() error {
	if err := setFocusAssist(1); err != nil {
		return fmt.Errorf("enable dnd: %w", err)
	}
	return nil
}

// DisableDND turns off Focus Assist (registry DWORD = 0).
func DisableDND() error {
	if err := setFocusAssist(0); err != nil {
		return fmt.Errorf("disable dnd: %w", err)
	}
	return nil
}

func setFocusAssist(value int) error {
	cmd := exec.Command("reg", "add", focusAssistKey,
		"/v", "Enabled",
		"/t", "REG_DWORD",
		"/d", fmt.Sprintf("%d", value),
		"/f",
	)
	return cmd.Run()
}
