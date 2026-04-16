//go:build integration

package blocker

/*
 * hosts_integration_test.go — touches the real /etc/hosts.
 *
 * Gated with the `integration` build tag so normal `go test ./...` skips it.
 * Run with:
 *
 *   sudo go test -tags=integration ./app/blocker/...
 *
 * WHY this exists: the unit tests in hosts_test.go cover the pure helpers
 * (stripSiloEntries, generateHostsEntries, isAllowed). The round-trip
 * Block → Restore path involves sudo, platform-specific paths, and DNS
 * flushing — hand-verifying is fragile and CI would need privileged runners.
 * Keeping it gated means we can still run it manually before shipping.
 */

import (
	"os"
	"strings"
	"testing"
)

func TestBlockRestore_RoundTrip(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("integration test requires root (sudo)")
	}

	// Snapshot hosts before we touch anything — fail-closed safety net.
	original, err := os.ReadFile(hostsFilePath())
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	t.Cleanup(func() {
		// Best-effort restore even if the test panicked.
		_ = os.WriteFile(hostsFilePath(), original, 0644)
	})

	h := NewHostsManager()
	if err := h.Block([]string{"test-silo-block.invalid"}, nil); err != nil {
		t.Fatalf("Block: %v", err)
	}

	// Verify the entry is actually in /etc/hosts.
	after, _ := os.ReadFile(hostsFilePath())
	if !strings.Contains(string(after), "test-silo-block.invalid") {
		t.Error("domain not present in /etc/hosts after Block")
	}
	if !strings.Contains(string(after), siloMarkerStart) {
		t.Error("silo marker not present in /etc/hosts")
	}

	if err := h.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	restored, _ := os.ReadFile(hostsFilePath())
	if strings.Contains(string(restored), "test-silo-block.invalid") {
		t.Error("domain still present after Restore")
	}
	if strings.Contains(string(restored), siloMarkerStart) {
		t.Error("silo marker still present after Restore")
	}
}
