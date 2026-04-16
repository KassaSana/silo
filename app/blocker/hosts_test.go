package blocker

import (
	"strings"
	"testing"
)

// stripSiloEntries is the cleanup function that removes silo's markers
// from an existing /etc/hosts. If it's wrong, we either leave garbage
// behind on unseal (breaking user's hosts file) or corrupt the user's
// own entries.

func TestStripSiloEntries_NoMarkersReturnsInput(t *testing.T) {
	input := "127.0.0.1 localhost\n::1 localhost\n"
	got := stripSiloEntries(input)
	// stripSiloEntries uses bufio.Scanner which splits on lines and we
	// rejoin with "\n" — trailing newline may be lost. The test should
	// compare semantically (line-set), not byte-exactly.
	assertSameLines(t, input, got)
}

func TestStripSiloEntries_RemovesBlock(t *testing.T) {
	input := strings.Join([]string{
		"127.0.0.1 localhost",
		siloMarkerStart,
		"127.0.0.1 twitter.com",
		"127.0.0.1 www.twitter.com",
		siloMarkerEnd,
		"# user comment",
	}, "\n")

	got := stripSiloEntries(input)

	if strings.Contains(got, "twitter.com") {
		t.Errorf("twitter.com should be removed:\n%s", got)
	}
	if strings.Contains(got, siloMarkerStart) || strings.Contains(got, siloMarkerEnd) {
		t.Errorf("markers should be removed:\n%s", got)
	}
	if !strings.Contains(got, "127.0.0.1 localhost") {
		t.Errorf("user entry should be preserved:\n%s", got)
	}
	if !strings.Contains(got, "# user comment") {
		t.Errorf("user comment should be preserved:\n%s", got)
	}
}

func TestStripSiloEntries_MultipleBlocks(t *testing.T) {
	// Defensive: if a prior run somehow left two silo blocks (e.g. from a
	// crash mid-write), strip should remove ALL of them, not just the first.
	input := strings.Join([]string{
		"127.0.0.1 localhost",
		siloMarkerStart,
		"127.0.0.1 twitter.com",
		siloMarkerEnd,
		"# middle",
		siloMarkerStart,
		"127.0.0.1 reddit.com",
		siloMarkerEnd,
	}, "\n")

	got := stripSiloEntries(input)

	if strings.Contains(got, "twitter.com") || strings.Contains(got, "reddit.com") {
		t.Errorf("both blocks should be stripped:\n%s", got)
	}
}

// generateHostsEntries builds the text written into /etc/hosts.
// This is where injection attacks land if Domain validation failed.
// The output MUST be well-formed per line.

func TestGenerateHostsEntries_WrapsWithMarkers(t *testing.T) {
	out := generateHostsEntries([]string{"twitter.com"})
	if !strings.HasPrefix(out, siloMarkerStart) {
		t.Error("output should start with siloMarkerStart")
	}
	if !strings.Contains(out, siloMarkerEnd) {
		t.Error("output should contain siloMarkerEnd")
	}
}

func TestGenerateHostsEntries_EmptyInputReturnsEmpty(t *testing.T) {
	if got := generateHostsEntries(nil); got != "" {
		t.Errorf("empty input should yield empty string, got %q", got)
	}
	if got := generateHostsEntries([]string{}); got != "" {
		t.Errorf("empty slice should yield empty string, got %q", got)
	}
}

func TestGenerateHostsEntries_AddsWwwVariant(t *testing.T) {
	out := generateHostsEntries([]string{"twitter.com"})
	if !strings.Contains(out, "127.0.0.1 twitter.com") {
		t.Errorf("missing bare domain:\n%s", out)
	}
	if !strings.Contains(out, "127.0.0.1 www.twitter.com") {
		t.Errorf("missing www variant:\n%s", out)
	}
}

func TestGenerateHostsEntries_DoesNotDoubleWww(t *testing.T) {
	// If input already starts with www., adding ANOTHER www. would produce
	// www.www.example.com — nonsense.
	out := generateHostsEntries([]string{"www.example.com"})
	if strings.Contains(out, "www.www.example.com") {
		t.Errorf("should not double-prefix www:\n%s", out)
	}
}

func TestGenerateHostsEntries_EveryLineIsWellFormed(t *testing.T) {
	// Regression guard: if validation ever slips, an injected newline in
	// a domain would create malformed lines. Every non-marker line must
	// start with "127.0.0.1 " and have no internal whitespace after that.
	out := generateHostsEntries([]string{"twitter.com", "reddit.com", "youtube.com"})
	for _, line := range strings.Split(out, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "127.0.0.1 ") {
			t.Errorf("malformed line: %q", line)
			continue
		}
		rest := strings.TrimPrefix(line, "127.0.0.1 ")
		if strings.ContainsAny(rest, " \t") {
			t.Errorf("line has unexpected whitespace after domain: %q", line)
		}
	}
}

// isAllowed is the filter that decides which distraction domains to skip.
// Wrong logic here = a site leaks through that should be blocked, or a site
// the user allowed is blocked anyway.

func TestIsAllowed_DirectMatch(t *testing.T) {
	set := map[string]bool{"github.com": true}
	if !isAllowed("github.com", set) {
		t.Error("direct match should return true")
	}
	if !isAllowed("GITHUB.COM", set) {
		t.Error("case-insensitive direct match should return true")
	}
}

func TestIsAllowed_WildcardPrefix(t *testing.T) {
	set := map[string]bool{"*.react.dev": true}
	if !isAllowed("react.dev", set) {
		t.Error("*.react.dev should match bare react.dev")
	}
	if !isAllowed("sub.react.dev", set) {
		t.Error("*.react.dev should match sub.react.dev")
	}
	if isAllowed("otherreact.dev", set) {
		t.Error("*.react.dev should NOT match otherreact.dev")
	}
}

func TestIsAllowed_PortWildcard(t *testing.T) {
	set := map[string]bool{"localhost:*": true}
	if !isAllowed("localhost:3000", set) {
		t.Error("localhost:* should match localhost:3000")
	}
	if !isAllowed("localhost", set) {
		// Prefix match: "localhost" alone has the prefix "localhost" — ambiguous.
		t.Log("localhost alone also matches localhost:* prefix — current behaviour")
	}
}

// Block + Restore round trip — a unit test that mocks the file boundary
// can't easily run here because writeHostsFile is platform-specific and
// uses the real /etc/hosts path. Integration coverage is deferred to
// hosts_integration_test.go (gated with //go:build integration).

// ── helpers ──

func assertSameLines(t *testing.T, want, got string) {
	t.Helper()
	w := splitLines(want)
	g := splitLines(got)
	if len(w) != len(g) {
		t.Errorf("line count mismatch: want %d, got %d\nwant: %q\ngot:  %q", len(w), len(g), want, got)
		return
	}
	for i := range w {
		if w[i] != g[i] {
			t.Errorf("line %d mismatch:\nwant: %q\ngot:  %q", i, w[i], g[i])
		}
	}
}

func splitLines(s string) []string {
	// Trim a single trailing newline so "a\n" and "a" compare equal.
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
