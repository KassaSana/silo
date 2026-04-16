//go:build darwin

package blocker

/*
 * process_darwin.go — macOS process enumeration and killing.
 *
 * Uses os/exec to call "ps" for listing processes — this is simpler
 * and more reliable than cgo bindings to sysctl. The overhead of
 * shelling out every 500ms is negligible.
 *
 * Killing uses syscall.Kill with SIGKILL (9) — the process doesn't
 * get a chance to handle it. This is intentional: we don't want the
 * blocked app to show "are you sure you want to quit?" dialogs.
 */

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// listProcesses returns all running user processes.
func listProcesses() ([]ProcessInfo, error) {
	// ps -eo pid,comm: list all processes with PID and command name
	// -o comm gives just the executable name (not full path with args)
	out, err := exec.Command("ps", "-eo", "pid,comm").Output()
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo
	lines := strings.Split(string(out), "\n")

	for _, line := range lines[1:] { // skip header
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split into PID and process name
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		pid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}

		name := strings.TrimSpace(parts[1])
		// ps -o comm may show full path like /Applications/Spotify.app/Contents/MacOS/Spotify
		// Extract just the binary name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}

		processes = append(processes, ProcessInfo{
			PID:  pid,
			Name: name,
		})
	}

	return processes, nil
}

// killProcess sends SIGKILL to a process.
func killProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
