//go:build windows

package blocker

/*
 * process_windows.go — Windows process enumeration and killing.
 *
 * Uses "tasklist" and "taskkill" commands for simplicity.
 * Post-MVP can switch to CreateToolhelp32Snapshot for better performance.
 */

import (
	"os/exec"
	"strconv"
	"strings"
)

func listProcesses() ([]ProcessInfo, error) {
	// tasklist /FO CSV /NH: list processes in CSV format, no header
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// CSV format: "process.exe","PID","Session Name","Session#","Mem Usage"
		parts := strings.Split(line, "\",\"")
		if len(parts) < 2 {
			continue
		}

		name := strings.Trim(parts[0], "\"")
		pidStr := strings.Trim(parts[1], "\"")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		processes = append(processes, ProcessInfo{
			PID:  pid,
			Name: name,
		})
	}

	return processes, nil
}

func killProcess(pid int) error {
	return exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid)).Run()
}
