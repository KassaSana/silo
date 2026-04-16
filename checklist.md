# Silo Build Checklist

## Phase 1: Skeleton (UI Foundation)
- [x] 1.1 Initialize Wails project with React+TS template
  - **Why:** Wails scaffolds the Go backend + React frontend + bridge between them. This gives us the desktop window shell to build inside.
  - **What we did:** Installed Wails v2.12.0, init'd with react-ts template, renamed from silo_init to silo, set window to 720x520 (min 600x400) with dark bg #0d1117.
- [x] 1.2 Terminal CSS foundation (colors, fonts, box-drawing)
  - **Why:** Every screen uses the same TUI aesthetic. Building the CSS system first means every component we create automatically looks right.
  - **What we did:** Created `terminal.css` with CSS variables for the full color palette, monospace font stack, 8px spacing grid, TUI component classes, and dark scrollbar theme.
- [x] 1.3 TUI component library (TuiBox, TuiHeader, TuiFooter, TuiList, TuiInput, ProgressBar)
  - **Why:** These are reusable building blocks. TuiBox = bordered container, TuiHeader = top bar with breadcrumb + clock, TuiFooter = keyboard shortcuts bar, TuiList = navigable list with ▸ indicator, TuiInput = text input.
  - **What we did:** 6 components in `frontend/src/components/`. Each is a simple, focused React component. TuiList uses generics (`<T>`) so it works with any item type.
- [x] 1.4 Dashboard screen with hardcoded data
  - **Why:** The dashboard is home base. Building it with fake data first lets us validate the layout and component system before wiring up real data.
  - **What we did:** Dashboard with workspace list (4 mock workspaces), focus stats with progress bars, and footer shortcuts. Screen router in App.tsx using state machine pattern instead of React Router.
- [x] 1.5 Keyboard navigation (useKeyboard + useNavigation hooks)
  - **Why:** This app is keyboard-first. j/k navigation, Enter to select, Esc to go back. The hook centralizes all keyboard handling so every screen gets it for free.
  - **What we did:** `useKeyboard` — global keydown listener that skips when user is in an input field (except Esc/Enter). `useNavigation` — wrapping j/k/arrow navigation for lists. Dashboard wired up with screen-specific keys (e/n/t/s).

## Phase 2: Workspaces (Data Layer)
- [x] 2.1 SQLite schema + migrations
  - **What:** `app/store/db.go` opens ~/.silo/silo.db with WAL mode. `migrations.go` creates tables (workspaces, sessions, daily_stats, settings) with IF NOT EXISTS so it's safe to run repeatedly. Default settings seeded with INSERT OR IGNORE.
- [x] 2.2 Workspace CRUD in Go
  - **What:** `app/workspace/workspace.go` — List, Get, Create, Update, Delete. Uses database/sql with parameterized queries (prevents SQL injection). JSON arrays stored as TEXT columns (pragmatic for small lists). UUID for primary keys.
- [x] 2.3 Templates (coding, studying, writing, research, leetcode, nuclear)
  - **What:** `app/workspace/templates.go` — 6 built-in templates matching design spec. Each defines apps + sites. "nuclear" template has empty arrays (blocks everything).
- [x] 2.4 Dashboard wired to real data
  - **What:** Dashboard now calls ListWorkspaces() from Go backend via Wails bindings. Shows empty state message when no workspaces exist. Stats still hardcoded (Phase 5).
- [x] 2.5 TemplatePicker screen
  - **What:** Two-phase flow: pick template → name workspace → creates via CreateFromTemplate(). Lists all templates with their apps/sites preview.
- [x] 2.6 WorkspaceEditor screen
  - **What:** Tabbed sections (apps/sites/obsidian) with [a]dd, [d]elete, Tab to switch sections. Supports both create-new and edit-existing modes. Ctrl+S to save.
- [x] 2.7 Obsidian vault/note config in editor
  - **What:** Obsidian section in WorkspaceEditor with vault name + note path inputs. Data flows through to SQLite via UpdateWorkspace/CreateWorkspace.

## Phase 3: Blocking Engine (OS-level enforcement)
- [x] 3.1 Hosts file read/write (cross-platform)
  - **What:** `hosts.go` shared logic (backup, block, restore, exception). `hosts_darwin.go` (macOS paths, dscacheutil flush, chflags protection). `hosts_windows.go` (Windows paths, ipconfig flush, icacls protection). Uses build tags for compile-time platform selection.
- [x] 3.2 DNS flush (cross-platform)
  - **What:** macOS: dscacheutil + mDNSResponder. Windows: ipconfig /flushdns. Called after every hosts modification.
- [x] 3.3 Curated distraction domain list (~500 domains)
  - **What:** `domains.go` — categorized list covering social media, news, entertainment, shopping, gaming, messaging, dating, sports, AI chat, adult content, crypto/stocks, gossip. Workspace allowed sites are excluded before writing to hosts.
- [x] 3.4 Process enumeration + system allowlist
  - **What:** `allowlist.go` — ~70 macOS system processes + ~30 Windows processes that must NEVER be killed. Fuzzy matching (case-insensitive substring) because process names vary. `process_darwin.go` uses `ps -eo pid,comm`. `process_windows.go` uses `tasklist`.
- [x] 3.5 Process monitor loop (500ms kill cycle)
  - **What:** `process.go` — goroutine with 500ms ticker. Each sweep: list all processes → check system allowlist → check workspace allowed → kill the rest. Logs killed processes for the UI. Uses SIGKILL on macOS, taskkill /F on Windows.
- [x] 3.6 Block page HTTP server on 127.0.0.1:9512
  - **What:** `blockpage.go` — serves styled "ACCESS DENIED" HTML. Reads Host header to show which domain was blocked. Shows workspace name, remaining time, task description. Tries port 80 first (elevated), falls back to 9512.
- [x] 3.7 Hosts file protection during seal
  - **What:** macOS: chflags schg (system immutable flag). Windows: icacls deny write. Removed on unseal or quick exception.

## Phase 4: Sessions (Core Loop)
- [x] 4.1 SealConfig screen with workspace summary
  - **What:** Shows allowed apps/sites, duration selector (15-min increments), lock type picker, activation ramp questions. 3-second countdown before sealing.
- [x] 4.2 Session creation in SQLite
  - **What:** `session.go` — full lifecycle manager. Writes session to SQLite on seal (crash recovery). Records workspace, task, lock, duration, exceptions, breach attempts. Updates daily_stats on completion.
- [x] 4.3 Countdown timer (Go goroutine -> Wails events)
  - **What:** `timer.go` — Go goroutine ticks every second, emits "timer:tick" event with remaining/elapsed/formatted. Emits "timer:done" on expiry. Frontend listens with EventsOn().
- [x] 4.4 Obsidian auto-open on seal
  - **What:** `workspace.OpenNote` launches `obsidian://open?vault=X&file=Y` via OS URL handler. Wired in `app/session/session.go:148`. No-op when vault or note is empty (opt-in).
- [x] 4.5 ActiveSeal screen with timer + blocked log
  - **What:** Big centered countdown, session info, blocked process log (polled every 2s). Completion phase with commit message input.
- [x] 4.6 QuickException screen + live allowlist update
  - **What:** 3-phase flow: choose type → enter value → type "i need this" to confirm. Updates hosts file in real-time for sites, adds to process allowlist for apps.
- [x] 4.7 Session completion (commit message + exception permanence prompt)
  - **What:** On timer done or unlock, shows commit message input. Records to session and updates daily_stats.
- [x] 4.8 Lock generation (crypto/rand)
  - **What:** `lock.go` — generates random a-zA-Z0-9 strings using crypto/rand (cryptographically secure). Character-by-character validation. Escalation logic built in.
- [x] 4.9 UnlockAttempt screen with char-by-char validation
  - **What:** Shows lock text in rows, textarea for typing, progress counter, escalation warning.
- [x] 4.10 Escalating lock (200 -> 400 -> 600)
  - **What:** Each failed attempt regenerates longer text. EscalateChars(attempt, 200, 200) = 200 + (attempt-1)*200.
- [x] 4.11 Obsidian daily note append on complete
  - **What:** `workspace.AppendToDailyNote` writes a markdown block to `<vault>/<note path>`, resolving `{date}` templates and auto-creating parent dirs. Wired in `app/session/session.go:390`. Covered by `app/workspace/obsidian_test.go`.

## Phase 5: Stats & Polish
- [x] 5.1 Stats queries + streak calculation
  - **What:** `app/stats/stats.go` — `GetSummary` (today/week/lifetime) and `calcStreak` with a one-day grace period. Unit tests in `app/stats/stats_test.go`.
- [x] 5.2 Stats screen with session history
  - **What:** `frontend/src/screens/Stats.tsx` — wired to real SQLite via `GetStatsSummary` + `GetRecentSessions`. Streak dots, progress bars, scrollable recent sessions.
- [x] 5.3 JSON export
  - **What:** `stats.ExportJSON` in `app/stats/stats.go:227`. Frontend copies to clipboard from `Stats.tsx:56`. Save-dialog upgrade listed as P2 in the implementation plan.
- [ ] 5.4 System tray icon — deferred to P1 (needs `getlantern/systray` dep + icon asset)
- [x] 5.5 DND integration
  - **What:** `app/platform/dnd_darwin.go` (Shortcuts app) + `dnd_windows.go`. Toggled from `session.go:155` (enable) and `:360` (disable), gated by the `dnd_integration` setting.
- [x] 5.6 Crash recovery from SQLite
  - **What:** `session.RecoverInterrupted` called from `app.go:42` on startup — marks any sessions left in `active` status as interrupted and cleans orphaned hosts entries via `blocker.CleanupOrphanedHosts`.

## Phase 6: Hardening
- [x] 6.0 Security boundary (P0 — 2026-04-16)
  - **What:** Input validation package at `app/validate/validate.go` wired into every Wails binding in `app.go` that accepts untrusted input. SQLite file perms locked to 0600 with dir at 0700 (`app/store/db.go`). Fail-closed hosts restore in `app/blocker/blocker.go` when `hosts.Block` errors. Unit tests cover injection vectors, escalation math, streak boundaries, Obsidian markdown, and template shape.
- [ ] 6.1 Cross-platform testing — Windows path untested on real hardware
- [ ] 6.2 Edge cases (VPN, custom DNS, multiple browsers, browser DoH bypass)
- [ ] 6.3 App icon + metadata — `wails.json` missing version/copyright; `build/darwin/Info.plist` still template
- [ ] 6.4 Installers (.dmg / .msi) — `build/windows/installer/project.nsi` is default NSIS
