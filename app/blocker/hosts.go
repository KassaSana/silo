package blocker

/*
 * hosts.go — Hosts file manipulation for website blocking.
 *
 * HOW IT WORKS:
 * 1. Read the original /etc/hosts and save a backup
 * 2. Append our blocked domains (pointing to 127.0.0.1)
 * 3. Flush DNS so the OS picks up the changes immediately
 * 4. On unseal: restore the original hosts file
 *
 * The blocked domains come from our curated distraction list MINUS
 * whatever the workspace allows. So if the workspace allows github.com,
 * it won't appear in the hosts block list.
 *
 * IMPORTANT from the design spec:
 * - Block page server MUST start BEFORE hosts file changes
 *   (otherwise blocked domains show connection errors)
 * - Quick exceptions update hosts file in real-time
 * - We protect the hosts file during seal to prevent manual edits
 *
 * Platform-specific paths and commands are in hosts_darwin.go / hosts_windows.go.
 * This file contains the shared logic.
 */

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const siloMarkerStart = "# >>> SILO BLOCK START — DO NOT EDIT <<<"
const siloMarkerEnd = "# >>> SILO BLOCK END <<<"

// HostsManager handles hosts file read/write/restore.
type HostsManager struct {
	originalContent string // backup of hosts file before we touched it
	blockedDomains  []string
}

// NewHostsManager creates a new hosts file manager.
func NewHostsManager() *HostsManager {
	return &HostsManager{}
}

// Block writes blocked domains to the hosts file.
// allowedSites are excluded from blocking.
// Returns error if hosts file can't be modified (permissions, etc.)
func (h *HostsManager) Block(distractionDomains, allowedSites []string) error {
	hostsPath := hostsFilePath()

	// 1. Read and backup original content
	original, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("read hosts file: %w", err)
	}
	h.originalContent = string(original)

	// 2. Build the block list: distractions MINUS allowed sites
	allowed := makeSet(allowedSites)
	var toBlock []string
	for _, domain := range distractionDomains {
		// Check both the domain and wildcard patterns
		if !isAllowed(domain, allowed) {
			toBlock = append(toBlock, domain)
		}
	}
	h.blockedDomains = toBlock

	// 3. Generate hosts entries
	blockEntries := generateHostsEntries(toBlock)

	// 4. Strip any existing silo entries, then append new ones
	cleaned := stripSiloEntries(h.originalContent)
	newContent := cleaned + "\n" + blockEntries

	// 5. Write the modified hosts file
	if err := writeHostsFile(hostsPath, newContent); err != nil {
		return fmt.Errorf("write hosts file: %w", err)
	}

	// 6. Flush DNS so changes take effect immediately
	if err := flushDNS(); err != nil {
		// Non-fatal — DNS will eventually update on its own
		fmt.Printf("warning: DNS flush failed: %v\n", err)
	}

	return nil
}

// AddException removes a domain from the block list (for quick exceptions).
func (h *HostsManager) AddException(domain string) error {
	hostsPath := hostsFilePath()

	content, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("read hosts file: %w", err)
	}

	// Remove just this domain from our block section
	lines := strings.Split(string(content), "\n")
	var newLines []string
	for _, line := range lines {
		// Skip lines that block this specific domain
		if strings.Contains(line, "127.0.0.1") && strings.Contains(line, domain) {
			continue
		}
		newLines = append(newLines, line)
	}

	if err := writeHostsFile(hostsPath, strings.Join(newLines, "\n")); err != nil {
		return err
	}

	return flushDNS()
}

// Restore puts the original hosts file back.
func (h *HostsManager) Restore() error {
	if h.originalContent == "" {
		// Nothing to restore — just strip any silo entries
		hostsPath := hostsFilePath()
		content, err := os.ReadFile(hostsPath)
		if err != nil {
			return err
		}
		cleaned := stripSiloEntries(string(content))
		if err := writeHostsFile(hostsPath, cleaned); err != nil {
			return err
		}
		return flushDNS()
	}

	if err := writeHostsFile(hostsFilePath(), h.originalContent); err != nil {
		return err
	}
	return flushDNS()
}

// ── Shared helpers ──

// generateHostsEntries creates the hosts file block section.
func generateHostsEntries(domains []string) string {
	if len(domains) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(siloMarkerStart + "\n")
	for _, domain := range domains {
		// Point blocked domains to localhost where our block page runs
		b.WriteString(fmt.Sprintf("127.0.0.1 %s\n", domain))
		// Also block www. variant
		if !strings.HasPrefix(domain, "www.") {
			b.WriteString(fmt.Sprintf("127.0.0.1 www.%s\n", domain))
		}
	}
	b.WriteString(siloMarkerEnd + "\n")
	return b.String()
}

// stripSiloEntries removes any previous silo block from the hosts content.
func stripSiloEntries(content string) string {
	var result []string
	inBlock := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if line == siloMarkerStart {
			inBlock = true
			continue
		}
		if line == siloMarkerEnd {
			inBlock = false
			continue
		}
		if !inBlock {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// makeSet converts a slice to a map for O(1) lookups.
func makeSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[strings.ToLower(item)] = true
	}
	return s
}

// isAllowed checks if a domain matches any allowed pattern.
// Supports wildcard: "localhost:*" matches "localhost:3000".
func isAllowed(domain string, allowed map[string]bool) bool {
	domain = strings.ToLower(domain)

	// Direct match
	if allowed[domain] {
		return true
	}

	// Check wildcard patterns (e.g. "localhost:*" or "*.react.dev")
	for pattern := range allowed {
		if strings.HasPrefix(pattern, "*.") {
			// *.react.dev matches react.dev and sub.react.dev
			suffix := pattern[1:] // ".react.dev"
			if domain == pattern[2:] || strings.HasSuffix(domain, suffix) {
				return true
			}
		}
		if strings.HasSuffix(pattern, ":*") {
			// localhost:* matches localhost:anything
			prefix := pattern[:len(pattern)-2] // "localhost"
			if strings.HasPrefix(domain, prefix) {
				return true
			}
		}
	}
	return false
}
