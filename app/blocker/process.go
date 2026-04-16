package blocker

/*
 * process.go — Process monitoring and killing.
 *
 * HOW IT WORKS:
 * 1. A goroutine runs every 500ms (the "kill loop")
 * 2. It enumerates all running processes
 * 3. For each process, it checks: system allowlist? workspace allowed? silo itself?
 * 4. If none match → kill it
 *
 * WHY 500ms? Fast enough that a user can't meaningfully use a blocked app
 * (it opens and dies within half a second). Slow enough that it doesn't
 * burn significant CPU.
 *
 * The kill loop returns a list of what it killed, which feeds the
 * "blocked just now" log on the ActiveSeal screen.
 *
 * Platform-specific process enumeration is in process_darwin.go / process_windows.go.
 */

import (
	"fmt"
	"sync"
	"time"
)

// ProcessInfo represents a running process.
type ProcessInfo struct {
	PID  int    `json:"pid"`
	Name string `json:"name"`
}

// BlockedAttempt records a killed process for the UI log.
type BlockedAttempt struct {
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
}

// ProcessMonitor watches for and kills non-allowed processes.
type ProcessMonitor struct {
	allowedApps []string
	stopCh      chan struct{}
	wg          sync.WaitGroup

	// mu protects blocked log
	mu      sync.Mutex
	blocked []BlockedAttempt
}

// NewProcessMonitor creates a process monitor for the given allowed apps.
func NewProcessMonitor(allowedApps []string) *ProcessMonitor {
	return &ProcessMonitor{
		allowedApps: allowedApps,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the kill loop in a goroutine.
// Returns immediately — the loop runs in the background.
func (pm *ProcessMonitor) Start() {
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		// Do an initial sweep immediately
		pm.sweep()

		for {
			select {
			case <-ticker.C:
				pm.sweep()
			case <-pm.stopCh:
				return
			}
		}
	}()
}

// Stop shuts down the kill loop.
func (pm *ProcessMonitor) Stop() {
	close(pm.stopCh)
	pm.wg.Wait()
}

// GetBlocked returns the list of recently blocked processes.
func (pm *ProcessMonitor) GetBlocked() []BlockedAttempt {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	// Return a copy
	result := make([]BlockedAttempt, len(pm.blocked))
	copy(result, pm.blocked)
	return result
}

// AddAllowedApp adds an app to the allowed list (for quick exceptions).
func (pm *ProcessMonitor) AddAllowedApp(app string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.allowedApps = append(pm.allowedApps, app)
}

// sweep checks all running processes and kills non-allowed ones.
func (pm *ProcessMonitor) sweep() {
	processes, err := listProcesses()
	if err != nil {
		fmt.Printf("warning: failed to list processes: %v\n", err)
		return
	}

	for _, proc := range processes {
		// Skip system processes (NEVER kill these)
		if IsSystemProcess(proc.Name) {
			continue
		}

		// Skip workspace-allowed apps
		if IsAllowedApp(proc.Name, pm.allowedApps) {
			continue
		}

		// This process is not allowed — kill it
		if err := killProcess(proc.PID); err != nil {
			// Some processes can't be killed (permission denied, zombie, etc.)
			// That's okay — we'll try again next sweep
			continue
		}

		// Log the kill
		pm.mu.Lock()
		pm.blocked = append(pm.blocked, BlockedAttempt{
			Name:      proc.Name,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		// Keep only last 50 entries
		if len(pm.blocked) > 50 {
			pm.blocked = pm.blocked[len(pm.blocked)-50:]
		}
		pm.mu.Unlock()
	}
}
