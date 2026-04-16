//go:build windows

package blocker

/*
 * hosts_windows.go — Windows-specific hosts file operations.
 *
 * Windows specifics:
 * - Hosts file at C:\Windows\System32\drivers\etc\hosts
 * - DNS flush: ipconfig /flushdns
 * - File protection: Set ACL to deny write access
 */

import (
	"fmt"
	"os"
	"os/exec"
)

func hostsFilePath() string {
	return `C:\Windows\System32\drivers\etc\hosts`
}

func writeHostsFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write hosts (may need admin): %w", err)
	}
	return nil
}

func flushDNS() error {
	return exec.Command("ipconfig", "/flushdns").Run()
}

func protectHostsFile() error {
	// Set read-only attribute via icacls
	return exec.Command("icacls", hostsFilePath(), "/deny", "Everyone:(W)").Run()
}

func unprotectHostsFile() error {
	return exec.Command("icacls", hostsFilePath(), "/remove:d", "Everyone").Run()
}
