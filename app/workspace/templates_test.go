package workspace

import (
	"strings"
	"testing"
)

// Templates are the primary UX onboarding path. If any of the six is
// malformed (missing name, empty-by-mistake, wrong shape) new users hit
// it immediately.

func TestBuiltinTemplates_AllPresent(t *testing.T) {
	want := []string{"coding", "studying", "writing", "research", "leetcode", "nuclear"}
	got := BuiltinTemplates()
	if len(got) != len(want) {
		t.Fatalf("got %d templates, want %d", len(got), len(want))
	}
	gotNames := make(map[string]bool)
	for _, tmpl := range got {
		gotNames[tmpl.Name] = true
	}
	for _, w := range want {
		if !gotNames[w] {
			t.Errorf("missing template %q", w)
		}
	}
}

func TestBuiltinTemplates_HaveDescription(t *testing.T) {
	for _, tmpl := range BuiltinTemplates() {
		if strings.TrimSpace(tmpl.Description) == "" {
			t.Errorf("template %q has empty description", tmpl.Name)
		}
	}
}

func TestBuiltinTemplates_NuclearIsEmpty(t *testing.T) {
	// "nuclear" blocks everything — both lists MUST be empty or it silently
	// becomes an ineffective "walk away" mode.
	for _, tmpl := range BuiltinTemplates() {
		if tmpl.Name != "nuclear" {
			continue
		}
		if len(tmpl.Apps) != 0 {
			t.Errorf("nuclear should have 0 apps, got %d", len(tmpl.Apps))
		}
		if len(tmpl.Sites) != 0 {
			t.Errorf("nuclear should have 0 sites, got %d", len(tmpl.Sites))
		}
	}
}

func TestBuiltinTemplates_NonNuclearHaveApps(t *testing.T) {
	// Every template except nuclear should allow at least one app —
	// otherwise the workspace is unusable.
	for _, tmpl := range BuiltinTemplates() {
		if tmpl.Name == "nuclear" {
			continue
		}
		if len(tmpl.Apps) == 0 {
			t.Errorf("template %q has no apps", tmpl.Name)
		}
	}
}

func TestBuiltinTemplates_CodingHasStackoverflowAndGithub(t *testing.T) {
	// Regression guard: the "coding" template's whole point is allowing
	// the developer's reference tools. Losing github or stackoverflow silently
	// would be a hostile change.
	coding := findTemplate(t, "coding")
	if !containsString(coding.Sites, "github.com") {
		t.Error("coding template missing github.com")
	}
	if !containsString(coding.Sites, "stackoverflow.com") {
		t.Error("coding template missing stackoverflow.com")
	}
}

func TestBuiltinTemplates_LeetcodeHasLeetcode(t *testing.T) {
	leet := findTemplate(t, "leetcode")
	if !containsString(leet.Sites, "leetcode.com") {
		t.Error("leetcode template missing leetcode.com")
	}
}

// ── helpers ──

func findTemplate(t *testing.T, name string) Template {
	t.Helper()
	for _, tmpl := range BuiltinTemplates() {
		if tmpl.Name == name {
			return tmpl
		}
	}
	t.Fatalf("template %q not found", name)
	return Template{}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
