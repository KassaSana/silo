package workspace

/*
 * templates.go — Pre-built workspace templates.
 *
 * CONCEPT: Templates lower the barrier to getting started. Instead of
 * manually adding apps and sites one by one, you pick "coding" and
 * get VS Code + Terminal + Chrome + the sites a developer needs.
 *
 * These match the design spec table exactly. Users start from a
 * template and customize — they can add/remove apps and sites later.
 *
 * The "nuclear" template is intentionally empty — it blocks EVERYTHING.
 * That's for when you need to walk away from the computer entirely.
 */

// Template defines a pre-built workspace configuration.
type Template struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Apps        []string `json:"apps"`
	Sites       []string `json:"sites"`
}

// BuiltinTemplates returns all available workspace templates.
func BuiltinTemplates() []Template {
	return []Template{
		{
			Name:        "coding",
			Description: "VS Code, Terminal, Chrome",
			Apps:        []string{"Visual Studio Code", "Terminal", "Google Chrome"},
			Sites:       []string{"localhost:*", "github.com", "stackoverflow.com", "claude.ai", "developer.mozilla.org"},
		},
		{
			Name:        "studying",
			Description: "Obsidian, Chrome, PDF reader",
			Apps:        []string{"Obsidian", "Google Chrome", "Preview"},
			Sites:       []string{"youtube.com", "claude.ai"},
		},
		{
			Name:        "writing",
			Description: "Obsidian, Chrome",
			Apps:        []string{"Obsidian", "Google Chrome"},
			Sites:       []string{"claude.ai", "scholar.google.com"},
		},
		{
			Name:        "research",
			Description: "Chrome, Obsidian, Notes",
			Apps:        []string{"Google Chrome", "Obsidian", "Notes"},
			Sites:       []string{"google.com", "scholar.google.com", "arxiv.org", "claude.ai"},
		},
		{
			Name:        "leetcode",
			Description: "VS Code, Terminal, Chrome",
			Apps:        []string{"Visual Studio Code", "Terminal", "Google Chrome"},
			Sites:       []string{"leetcode.com", "neetcode.io", "claude.ai", "en.cppreference.com"},
		},
		{
			Name:        "nuclear",
			Description: "nothing allowed — walk away",
			Apps:        []string{},
			Sites:       []string{},
		},
	}
}
