package validate

import (
	"strings"
	"testing"
)

// Domain — the hosts-injection test is the whole point of this package.
// If this test regresses, an attacker can corrupt /etc/hosts by naming
// a workspace site.

func TestDomain_Valid(t *testing.T) {
	cases := []string{
		"example.com",
		"sub.example.com",
		"a.b.c.d.example.com",
		"example.com:8080",
		"example.com:1",
		"example.com:65535",
		"*.example.com",
		"a-b.example.com",
		"xn--bcher-kva.example", // punycode
		"1.example.com",         // numeric label OK (not a full IP)
	}
	for _, c := range cases {
		if err := Domain(c); err != nil {
			t.Errorf("Domain(%q) rejected valid input: %v", c, err)
		}
	}
}

func TestDomain_InjectionRejected(t *testing.T) {
	// The headline test: each of these, if accepted, would inject arbitrary
	// content into /etc/hosts via fmt.Sprintf("127.0.0.1 %s\n", domain).
	cases := map[string]string{
		"newline":           "evil.com\n127.0.0.1 bank.com",
		"crlf":              "evil.com\r\n127.0.0.1 bank.com",
		"null":              "evil.com\x00",
		"tab":               "evil.com\t",
		"space in middle":   "evil .com",
		"leading space":     " evil.com",
		"trailing space":    "evil.com ",
		"semicolon":         "evil.com;rm -rf /",
		"pipe":              "evil.com|nc",
		"ampersand":         "evil.com&",
		"hash comment":      "evil.com # comment",
		"shell substitute":  "evil.com$(id)",
		"backtick":          "evil.com`id`",
		"quote":             "evil.com\"",
	}
	for name, c := range cases {
		if err := Domain(c); err == nil {
			t.Errorf("Domain(%q) [%s] should have been rejected", c, name)
		}
	}
}

func TestDomain_Malformed(t *testing.T) {
	cases := map[string]string{
		"empty":            "",
		"just dot":         ".",
		"leading dot":      ".example.com",
		"trailing dot":     "example.com.",
		"double dot":       "foo..bar.com",
		"single label":     "localhost",
		"label too long":   strings.Repeat("a", 64) + ".com",
		"total too long":   strings.Repeat("a.", 130) + "com",
		"hyphen start":     "-foo.com",
		"hyphen end":       "foo-.com",
		"port zero":        "example.com:0",
		"port too big":     "example.com:70000",
		"port non-numeric": "example.com:abc",
		"ipv4 literal":     "192.168.1.1",
		"ipv4 with port":   "192.168.1.1:80",
		"ipv6 literal":     "::1",
		"bracketed v6":     "[::1]:80",
		"wildcard middle":  "foo.*.com",
		"wildcard only":    "*",
		"wildcard alone":   "*.",
		"non-ascii":        "münchen.de",
		"underscore":       "foo_bar.com", // RFC 1035 forbids underscores in hostnames
	}
	for name, c := range cases {
		if err := Domain(c); err == nil {
			t.Errorf("Domain(%q) [%s] should have been rejected", c, name)
		}
	}
}

// Text — the workspace-name / task-description sanitiser.

func TestText_StripsControlCharsInMiddle(t *testing.T) {
	// A task description with a newline mid-string would break the Obsidian
	// daily-note markdown. We strip rather than reject, because pasted content
	// commonly carries stray control chars.
	cleaned, err := Text("build\nauth flow", MaxTextLen)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.ContainsAny(cleaned, "\n\r\x00") {
		t.Errorf("control chars not stripped: %q", cleaned)
	}
	if !strings.Contains(cleaned, "build") || !strings.Contains(cleaned, "auth flow") {
		t.Errorf("content lost during strip: %q", cleaned)
	}
}

func TestText_TrimsWhitespace(t *testing.T) {
	cleaned, err := Text("  hello  ", MaxTextLen)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleaned != "hello" {
		t.Errorf("trim failed: %q", cleaned)
	}
}

func TestText_RejectsEmptyAfterStrip(t *testing.T) {
	cases := []string{"", "   ", "\n\n", "\x00\x00", "\t"}
	for _, c := range cases {
		if _, err := Text(c, MaxTextLen); err == nil {
			t.Errorf("Text(%q) should be rejected as empty", c)
		}
	}
}

func TestText_RejectsOverlong(t *testing.T) {
	long := strings.Repeat("a", MaxTextLen+1)
	if _, err := Text(long, MaxTextLen); err == nil {
		t.Error("overlong input should be rejected")
	}
}

func TestText_RejectsInvalidUTF8(t *testing.T) {
	// Invalid UTF-8 byte sequence
	if _, err := Text(string([]byte{0xff, 0xfe}), MaxTextLen); err == nil {
		t.Error("invalid UTF-8 should be rejected")
	}
}

func TestText_PreservesUnicode(t *testing.T) {
	// Unicode content is fine — we only strip control categories.
	cleaned, err := Text("café résumé 日本語", MaxTextLen)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cleaned != "café résumé 日本語" {
		t.Errorf("unicode mangled: %q", cleaned)
	}
}

// AppName — process-name matcher input.

func TestAppName_RejectsPathSeparators(t *testing.T) {
	cases := []string{
		"/usr/bin/firefox",
		"..\\windows\\system32",
		"firefox/helper",
	}
	for _, c := range cases {
		if _, err := AppName(c); err == nil {
			t.Errorf("AppName(%q) should be rejected", c)
		}
	}
}

func TestAppName_RejectsShellMeta(t *testing.T) {
	cases := []string{"firefox; rm -rf", "firefox$PATH", "firefox|nc", "firefox&", "firefox>log"}
	for _, c := range cases {
		if _, err := AppName(c); err == nil {
			t.Errorf("AppName(%q) should be rejected", c)
		}
	}
}

func TestAppName_AcceptsNormalNames(t *testing.T) {
	cases := []string{"Firefox", "Google Chrome", "code-insiders", "VS Code.app", "obsidian"}
	for _, c := range cases {
		if _, err := AppName(c); err != nil {
			t.Errorf("AppName(%q) rejected: %v", c, err)
		}
	}
}

// Path — Obsidian vault path / daily note path.

func TestPath_AllowsEmpty(t *testing.T) {
	// Obsidian is opt-in; empty path means "not configured".
	if p, err := Path(""); err != nil || p != "" {
		t.Errorf("empty path should be allowed: %q, %v", p, err)
	}
}

func TestPath_RejectsControlChars(t *testing.T) {
	cases := []string{"vault\x00path", "vault\npath", "vault\rpath"}
	for _, c := range cases {
		if _, err := Path(c); err == nil {
			t.Errorf("Path(%q) should be rejected", c)
		}
	}
}

func TestPath_AllowsNormalPaths(t *testing.T) {
	cases := []string{
		"~/Documents/ObsidianVault",
		"/Users/me/vault",
		"C:\\Users\\me\\vault",
		"daily/{date}.md",
		"daily",
	}
	for _, c := range cases {
		if _, err := Path(c); err != nil {
			t.Errorf("Path(%q) rejected: %v", c, err)
		}
	}
}
