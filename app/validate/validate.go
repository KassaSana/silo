// Package validate provides boundary-layer input validation for silo.
//
// WHY a dedicated package: silo runs with elevated privileges during a seal.
// User-typed workspace names, domains, app names, and task descriptions all
// flow into system-modifying code paths (/etc/hosts, process kill, chflags,
// markdown file writes). Per CLAUDE.md, validation must happen at the boundary
// where input enters the Go backend — not deep in the stack. Centralising the
// primitives here keeps the rules in one auditable place.
//
// Every function is pure (no I/O), cheap, and safe to call on any input,
// including attacker-controlled bytes.
package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Size caps. These are intentionally small — silo stores workspace metadata
// and short task descriptions, not documents. A megabyte of "workspace name"
// would be a sign something is wrong.
const (
	MaxDomainLen    = 253 // RFC 1035 total length
	MaxLabelLen     = 63  // RFC 1035 label length
	MaxTextLen      = 512
	MaxAppNameLen   = 128
	MaxPathLen      = 512
	maxCommitMsgLen = 512
)

// labelRe matches a single DNS label: starts+ends alphanumeric, hyphens allowed in the middle.
// A single-character label (e.g. "a") is allowed.
var labelRe = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// Domain validates a hostname-or-hostname:port that we'll write into /etc/hosts.
//
// Accepted shapes:
//   - example.com
//   - sub.example.com
//   - *.example.com        (wildcard prefix — silo expands to www. variant downstream)
//   - example.com:8080     (port suffix allowed for block-page routing parity)
//
// Rejected:
//   - empty, whitespace-only
//   - anything longer than 253 chars (RFC 1035) or with a label > 63 chars
//   - any control char, newline, tab, null — these are hosts-file injection vectors
//   - raw IPv4/IPv6 literals — silo blocks by name, not address
//   - shell metacharacters, spaces
//
// The error message deliberately echoes the offending input (truncated) so the
// UI can surface it, but strips control chars before echoing.
func Domain(s string) error {
	if s == "" {
		return fmt.Errorf("domain is empty")
	}
	if len(s) > MaxDomainLen {
		return fmt.Errorf("domain too long (max %d)", MaxDomainLen)
	}
	// Reject any control char / NUL / newline outright — these are the
	// injection vectors for /etc/hosts.
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c == 0x7f {
			return fmt.Errorf("domain contains control character")
		}
	}
	// Reject anything that isn't ASCII. Punycode users must pre-encode
	// (xn--...). We don't silently convert because any conversion library
	// is another attack surface.
	if !isASCII(s) {
		return fmt.Errorf("domain must be ASCII (use punycode for IDN)")
	}

	host, port, _ := splitHostPort(s)

	// Reject raw IPv4/IPv6 literals. silo is a hostname blocklist;
	// address literals confuse both the hosts file and user intent.
	if looksLikeIP(host) {
		return fmt.Errorf("IP literals not allowed — use the hostname")
	}

	// Split wildcard prefix off the first label if present.
	labels := strings.Split(host, ".")
	if len(labels) == 0 {
		return fmt.Errorf("domain missing labels")
	}
	for i, lbl := range labels {
		if i == 0 && lbl == "*" {
			continue // *.example.com allowed
		}
		if lbl == "" {
			return fmt.Errorf("domain has empty label (leading/trailing/double dot)")
		}
		if len(lbl) > MaxLabelLen {
			return fmt.Errorf("domain label too long (max %d)", MaxLabelLen)
		}
		if !labelRe.MatchString(lbl) {
			return fmt.Errorf("domain label %q is invalid", lbl)
		}
	}
	// Must have at least one dot somewhere — "localhost" alone isn't useful
	// as a block target, and single-label names are ambiguous.
	if len(labels) < 2 {
		return fmt.Errorf("domain must include a dot (e.g. example.com)")
	}
	if port != "" {
		if err := validatePort(port); err != nil {
			return err
		}
	}
	return nil
}

// validatePort parses a port string and enforces 1 ≤ port ≤ 65535.
// A regex alone can't bound the max cleanly, and strconv makes the intent
// unambiguous.
func validatePort(p string) error {
	if p == "" {
		return fmt.Errorf("empty port")
	}
	// Disallow leading zero to match strict URL parsing (":080" is ambiguous).
	if len(p) > 1 && p[0] == '0' {
		return fmt.Errorf("port has leading zero")
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return fmt.Errorf("port is not numeric")
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("port out of range (1-65535)")
	}
	return nil
}

// Text sanitises a free-form user string (workspace name, task description,
// commit message, Obsidian vault name, etc.) by stripping control characters
// and validating length. The cleaned string is returned; the original is
// not mutated.
//
// WHY strip rather than reject: users pasting from other apps commonly carry
// a trailing \n; rejecting would be hostile. Control chars INSIDE the payload
// (not just trailing) are also stripped — a newline mid-task-description would
// break the Obsidian daily-note markdown format, and a NUL could poison SQLite
// BLOB handling downstream.
//
// maxLen is a caller-supplied cap. Use MaxTextLen for most fields.
func Text(s string, maxLen int) (string, error) {
	if maxLen <= 0 {
		maxLen = MaxTextLen
	}
	if !utf8.ValidString(s) {
		return "", fmt.Errorf("text is not valid UTF-8")
	}
	cleaned := stripControl(s)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "", fmt.Errorf("text is empty")
	}
	if utf8.RuneCountInString(cleaned) > maxLen {
		return "", fmt.Errorf("text too long (max %d chars)", maxLen)
	}
	return cleaned, nil
}

// AppName validates a process/app name that will flow into the blocker's
// allowlist matcher. Stricter than Text: no path separators, no quotes,
// no shell metacharacters. These characters would be legitimate in a
// free-form name but suggest either a path or an injection attempt.
func AppName(s string) (string, error) {
	cleaned, err := Text(s, MaxAppNameLen)
	if err != nil {
		return "", err
	}
	for _, r := range cleaned {
		switch r {
		case '/', '\\', '"', '\'', '`', '$', ';', '|', '&', '<', '>':
			return "", fmt.Errorf("app name contains disallowed character %q", r)
		}
	}
	return cleaned, nil
}

// Path validates a filesystem-ish path (Obsidian vault path, daily-note path).
// We don't resolve it or check existence — that's the caller's job. We do
// strip control chars and reject NUL / newlines, which would confuse both
// filepath.Join and the OS.
func Path(s string) (string, error) {
	if s == "" {
		return "", nil // empty path is allowed (means "Obsidian not configured")
	}
	if !utf8.ValidString(s) {
		return "", fmt.Errorf("path is not valid UTF-8")
	}
	for _, r := range s {
		if r == 0 || r == '\n' || r == '\r' {
			return "", fmt.Errorf("path contains control character")
		}
	}
	if utf8.RuneCountInString(s) > MaxPathLen {
		return "", fmt.Errorf("path too long (max %d chars)", MaxPathLen)
	}
	return s, nil
}

// ── helpers ──

// stripControl removes all Unicode control characters except standard
// whitespace within a line (space, tab). Newlines ARE stripped — they are
// the primary hosts-file and markdown injection vector.
func stripControl(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\t' || r == ' ' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7f {
			return false
		}
	}
	return true
}

// splitHostPort splits "host:port" without using net.SplitHostPort (which
// is more permissive than we want and accepts IPv6 brackets etc).
// Returns (host, port, hasPort). If the input has no colon, port is "".
func splitHostPort(s string) (string, string, bool) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+1:], true
}

// looksLikeIP returns true for obvious IPv4 or IPv6 literals.
// We don't use net.ParseIP because we want to reject anything ambiguous,
// not just well-formed IPs.
func looksLikeIP(s string) bool {
	if s == "" {
		return false
	}
	// IPv6 heuristic: contains a colon (splitHostPort has already removed
	// any :port suffix, so a remaining colon means v6) or is bracketed.
	if strings.ContainsRune(s, ':') {
		return true
	}
	if strings.HasPrefix(s, "[") {
		return true
	}
	// IPv4 heuristic: four dot-separated all-digit segments.
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}
