# silo — Claude Code quickstart

> Read `SILO_DESIGN.md` first. It's the full product spec.

## What you're building

A cross-platform (macOS + Windows) desktop focus app that uses a **workspace model** — you define the exact apps and sites you need for a task, and everything else is blocked at the OS level. Terminal-aesthetic UI. Built with **Go** (backend) + **React/TypeScript** (frontend) + **Wails v2** (desktop framework) + **SQLite** (local storage).

## The mental model

This is NOT a blocklist app. It's a workspace app. The user defines what's ALLOWED (5 apps, 8 sites), and everything else in the universe is blocked. This is the opposite of Cold Turkey / Freedom / etc.

## MVP scope

Ship **workspace-first lockdown** with these features:

1. **Workspaces** — named configs with allowed_apps, allowed_sites, optional Obsidian vault/note link. Pre-built templates (coding, studying, writing, research, leetcode, nuclear).
2. **OS-level blocking** — hosts file for websites (curated ~500 domain distraction list, excluding workspace's allowed sites), process killing for apps (500ms poll, kill non-allowed, protect system processes).
3. **Quick exceptions** — friction-gated temporary allowances during active seal. Type domain → type "i need this" → session-scoped, logged.
4. **Session lifecycle** — activation ramp (task + first step), countdown timer, commit message on completion, crash recovery from SQLite.
5. **Lock system** — random-text (escalating 200→400→600), timer (no override), reboot (restart required).
6. **Obsidian integration** — auto-open linked note on seal via `obsidian://` URI, append session log to daily note on completion.
7. **Stats** — session history, daily/weekly focus time, streak counter, JSON export.

## Build order

1. **Skeleton** — Wails init, terminal CSS, TUI components, Dashboard with hardcoded data, keyboard nav
2. **Workspaces** — SQLite, workspace CRUD, templates, WorkspaceEditor, Obsidian config
3. **Blocking engine** — hosts file, DNS flush, distraction domain list, process monitor/kill, system allowlist, block page server
4. **Sessions** — SealConfig, timer, Obsidian auto-open, ActiveSeal, QuickException, locks, UnlockAttempt, daily note append
5. **Stats** — queries, Stats screen, export, tray icon, DND integration
6. **Hardening** — cross-platform testing, installers

## Critical notes

- **Workspace = whitelist.** Block everything EXCEPT allowed. For MVP: curated distraction list in hosts file minus allowed sites. Post-MVP: firewall-level true whitelist.
- **Go build tags** for platform code (`//go:build darwin` / `//go:build windows`). No `runtime.GOOS` inline checks.
- **Session state survives crashes.** Write to SQLite on seal, restore on launch.
- **Block page server starts BEFORE hosts file changes.** Otherwise blocked domains show connection errors.
- **System process allowlist is CRITICAL.** Never kill Finder, explorer.exe, etc.
- **Quick exceptions update hosts file in real-time** — remove domain from blocklist, flush DNS, persist to session.
- **Obsidian is just URI launch + file append.** `open "obsidian://open?vault=X&file=Y"` on macOS, `start` on Windows.
- **Lock text:** `crypto/rand`, charset `a-zA-Z0-9`, stored in session record.
- **Frontend:** no animations, no transitions, instant screen swaps. Monospace everything.

## Key files

- `SILO_DESIGN.md` — full spec: architecture, all 9 screen mockups, SQLite schema, blocking mechanisms, rationale
- `SILO_DESIGN.md` > "UI specification" — screen-by-screen ASCII mockups
- `SILO_DESIGN.md` > "Data model" — complete SQLite schema
- `SILO_DESIGN.md` > "File structure" — expected project layout
- `SILO_DESIGN.md` > "Training mode" — planned post-MVP keyboard drill feature (design now, build later)