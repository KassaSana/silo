package blocker

/*
 * blocker.go — The blocking engine coordinator.
 *
 * CONCEPT: This is the orchestrator. When a session seals, it:
 *   1. Starts the block page server (MUST be first)
 *   2. Modifies the hosts file (blocking websites)
 *   3. Starts the process monitor (killing non-allowed apps)
 *
 * On unseal, it reverses everything in the opposite order:
 *   1. Stop process monitor
 *   2. Restore hosts file
 *   3. Stop block page server
 *
 * The order matters! If we modify hosts before the block page is up,
 * users see "connection refused". If we stop the block page before
 * restoring hosts, same problem.
 *
 * Quick exceptions flow through here: AddSiteException updates the
 * hosts file in real-time, AddAppException tells the process monitor.
 */

import "fmt"

// Engine coordinates all blocking mechanisms.
type Engine struct {
	hosts     *HostsManager
	processes *ProcessMonitor
	blockPage *BlockPageServer

	running bool
}

// NewEngine creates a new blocking engine.
func NewEngine() *Engine {
	return &Engine{
		hosts: NewHostsManager(),
	}
}

// Engage activates all blocking for a workspace.
// This is called when a session is sealed.
func (e *Engine) Engage(allowedApps, allowedSites []string, workspaceName, taskDesc string) error {
	if e.running {
		return fmt.Errorf("blocking engine already running")
	}

	// 1. Start block page server FIRST
	e.blockPage = NewBlockPageServer(workspaceName, taskDesc)
	if err := e.blockPage.Start(); err != nil {
		return fmt.Errorf("start block page: %w", err)
	}

	// 2. Modify hosts file (blocks websites)
	distractions := DistractionDomains()
	if err := e.hosts.Block(distractions, allowedSites); err != nil {
		// Rollback: stop block page
		e.blockPage.Stop()
		return fmt.Errorf("block websites: %w", err)
	}

	// 3. Protect the hosts file from manual edits
	if err := protectHostsFile(); err != nil {
		fmt.Printf("warning: could not protect hosts file: %v\n", err)
	}

	// 4. Start process monitor (kills non-allowed apps)
	e.processes = NewProcessMonitor(allowedApps)
	e.processes.Start()

	e.running = true
	return nil
}

// Disengage stops all blocking and restores the system.
// This is called when a session completes or is unlocked.
func (e *Engine) Disengage() error {
	if !e.running {
		return nil
	}

	// Reverse order of engagement

	// 1. Stop process monitor
	if e.processes != nil {
		e.processes.Stop()
	}

	// 2. Unprotect and restore hosts file
	unprotectHostsFile()
	if err := e.hosts.Restore(); err != nil {
		fmt.Printf("warning: failed to restore hosts: %v\n", err)
	}

	// 3. Stop block page server
	if e.blockPage != nil {
		e.blockPage.Stop()
	}

	e.running = false
	return nil
}

// AddSiteException allows a blocked site during the current session.
func (e *Engine) AddSiteException(domain string) error {
	if !e.running {
		return fmt.Errorf("no active seal")
	}

	// Remove from hosts file and flush DNS
	unprotectHostsFile()
	if err := e.hosts.AddException(domain); err != nil {
		protectHostsFile()
		return err
	}
	protectHostsFile()
	return nil
}

// AddAppException allows a blocked app during the current session.
func (e *Engine) AddAppException(app string) {
	if e.processes != nil {
		e.processes.AddAllowedApp(app)
	}
}

// GetBlockedAttempts returns recent blocked process attempts for the UI.
func (e *Engine) GetBlockedAttempts() []BlockedAttempt {
	if e.processes != nil {
		return e.processes.GetBlocked()
	}
	return nil
}

// UpdateTimeRemaining updates the countdown on the block page.
func (e *Engine) UpdateTimeRemaining(remaining string) {
	if e.blockPage != nil {
		e.blockPage.SetTimeRemaining(remaining)
	}
}

// IsRunning returns whether blocking is currently active.
func (e *Engine) IsRunning() bool {
	return e.running
}

// CleanupOrphanedHosts strips any leftover silo block markers from the
// hosts file. Used on startup in case a prior session crashed mid-seal
// and left its block list in place. Safe no-op if nothing to clean.
func CleanupOrphanedHosts() error {
	unprotectHostsFile() // no-op if already unprotected
	h := NewHostsManager()
	// Restore with no backup just strips silo markers + flushes DNS.
	return h.Restore()
}
