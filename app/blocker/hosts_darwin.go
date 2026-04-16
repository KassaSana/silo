//go:build darwin

package blocker

/*
 * hosts_darwin.go — macOS-specific hosts file operations.
 *
 * BUILD TAG: The "//go:build darwin" line at the top tells the Go compiler
 * "only include this file when building for macOS." The Windows version
 * (hosts_windows.go) has "//go:build windows". The compiler picks ONE
 * based on the target platform. This is how Go handles cross-platform
 * code without #ifdef or runtime checks.
 *
 * macOS specifics:
 * - Hosts file at /etc/hosts (needs root/sudo to write)
 * - DNS flush: dscacheutil -flushcache && killall -HUP mDNSResponder
 * - File protection: chflags schg (set the system immutable flag)
 */

import (
	"fmt"
	"os"
	"os/exec"
)

// hostsFilePath returns the macOS hosts file location.
func hostsFilePath() string {
	return "/etc/hosts"
}

// writeHostsFile writes content to the hosts file.
// On macOS, /etc/hosts requires elevated privileges.
func writeHostsFile(path, content string) error {
	// Remove immutable flag first if it was set
	exec.Command("chflags", "noschg", path).Run()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write hosts (may need sudo): %w", err)
	}
	return nil
}

// flushDNS clears the macOS DNS cache so hosts changes take effect.
// Without this, the browser would keep using cached DNS for ~60 seconds.
func flushDNS() error {
	// macOS has two DNS caches that both need flushing:
	// 1. dscacheutil — the directory service cache
	if err := exec.Command("dscacheutil", "-flushcache").Run(); err != nil {
		return fmt.Errorf("flush dscacheutil: %w", err)
	}
	// 2. mDNSResponder — the mDNS daemon cache
	if err := exec.Command("killall", "-HUP", "mDNSResponder").Run(); err != nil {
		return fmt.Errorf("flush mDNSResponder: %w", err)
	}
	return nil
}

// protectHostsFile makes the hosts file immutable so it can't be
// manually edited while silo is sealed. Uses macOS chflags.
func protectHostsFile() error {
	return exec.Command("chflags", "schg", hostsFilePath()).Run()
}

// unprotectHostsFile removes the immutable flag.
func unprotectHostsFile() error {
	return exec.Command("chflags", "noschg", hostsFilePath()).Run()
}
