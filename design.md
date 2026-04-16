# silo

> OS-level focus enforcement for people who can't trust themselves to stay on task.

## What is this?

silo is a cross-platform (macOS + Windows) desktop application that enforces deep focus at the operating system level. You define a **workspace** — the exact set of apps and sites you need for a task — and silo blocks everything else. Once a session is engaged ("sealed"), it cannot be trivially undone.

The app has a minimal, terminal-inspired UI driven primarily by keyboard shortcuts.

This is not a wellness app. This is a lock on the door.

---

## Why this exists (design rationale)

The target user has a brain that:

- Takes **45-60 minutes** to recover focus after a single context switch (vs 23 min for neurotypical brains)
- Is an **expert workaround-finder** — a blocklist of 50 sites still leaves the entire rest of the internet
- Struggles with **task activation**, not task execution — starting is the hard part
- Responds to **artificial urgency** — visible countdowns and time pressure generate dopamine
- Benefits from **environmental design over willpower** — making distraction physically impossible
- Needs **visual progress feedback** — streaks, session logs, and completed blocks provide reward signals
- Loses flow to **mouse-driven context switching** — keyboard muscle memory keeps transitions frictionless

Every design decision flows from these constraints.

---

## Core concept: workspace-first blocking

### The problem with blocklists

Traditional blockers use a **blocklist model**: you list what's bad, everything else is allowed. Fatal flaw: you can never block everything. Block Twitter, brain finds Reddit. Block Reddit, discovers Hacker News. The ADHD brain is infinitely creative at finding new distraction vectors.

### The workspace model

silo inverts this. Instead of defining what's bad, you define what the work IS:

    workspace: react-project
      allowed apps:   VS Code, Terminal, Chrome
      allowed sites:  localhost:*, react.dev, claude.ai, github.com, stackoverflow.com
      blocked:        EVERYTHING ELSE
      obsidian:       vault=CS, note=projects/react-project-log

When you seal this workspace, only the listed apps and sites work. Everything else is blocked at the OS level. Distraction #47 that you haven't discovered yet is already blocked.

### The escape valve: quick exceptions

Risk: you seal, then realize you need docs.python.org and it's not in your workspace.

Solution: **quick exceptions** — a friction-gated mechanism to temporarily add a site or app during an active seal.

- Press [x] during active seal
- Type the domain or app name
- Type confirmation phrase "i need this" (friction barrier — stops impulse, not genuine need)
- Exception lasts for THIS session only
- All exceptions logged in session history
- Post-session prompt: "Add these exceptions permanently?"

### Workspace templates

Pre-built templates reduce setup friction:

| Template | Allowed apps | Allowed sites |
|---|---|---|
| coding | VS Code, Terminal, Chrome | localhost:*, github.com, stackoverflow.com, claude.ai, MDN |
| studying | Obsidian, Chrome, PDF reader | youtube.com, claude.ai (+ user adds course sites) |
| writing | Obsidian, Chrome | claude.ai, google scholar |
| research | Chrome, Obsidian, Notes | google.com, scholar.google.com, arxiv.org, claude.ai |
| leetcode | VS Code, Terminal, Chrome | leetcode.com, neetcode.io, claude.ai, cppreference.com |
| nuclear | (none) | (none) — walk away from the computer |

Users start from a template and customize.

---

## Architecture overview

    ┌─────────────────────────────────────────────────────────────┐
    │                         silo                                 │
    │                                                             │
    │  ┌───────────────┐    ┌──────────────────┐                  │
    │  │   React UI    │◄──►│   Wails Bridge   │                  │
    │  │  (frontend)   │    │   (Go ↔ JS)      │                  │
    │  └───────────────┘    └──────┬───────────┘                  │
    │                              │                              │
    │                   ┌──────────▼───────────┐                  │
    │                   │   Go Backend Core    │                  │
    │                   │                      │                  │
    │                   │  - Block Engine       │                  │
    │                   │    (hosts, firewall,  │                  │
    │                   │     process mgr)      │                  │
    │                   │  - Session Manager    │                  │
    │                   │    (timer, locks,     │                  │
    │                   │     stats)            │                  │
    │                   │  - Workspace Manager  │                  │
    │                   │    (SQLite, templates, │                  │
    │                   │     obsidian)          │                  │
    │                   └──────────────────────┘                  │
    │                                                             │
    │  ┌───────────────────────────────────────────────────────┐  │
    │  │   OS Layer (elevated privileges)                      │  │
    │  │   - /etc/hosts (mac) / hosts (win)                    │  │
    │  │   - pf firewall (mac) / WFP (win)                    │  │
    │  │   - process monitoring & killing                      │  │
    │  │   - self-protection (watchdog) — post-MVP             │  │
    │  └───────────────────────────────────────────────────────┘  │
    └─────────────────────────────────────────────────────────────┘

### Tech stack

| Layer | Technology | Why |
|---|---|---|
| Desktop framework | Wails v2 | Go backend + web frontend. Lightweight, no bundled Chromium. |
| Backend | Go | Cross-platform OS APIs, process management, single binary. |
| Frontend | React + TypeScript | TUI-style screens. Plain CSS, no Tailwind, no component libraries. |
| Storage | SQLite (modernc.org/sqlite, pure Go) | Local-only, privacy-first. |
| CLI (post-MVP) | cobra | Optional CLI: silo seal, silo status, etc. |

### Cross-platform blocking mechanisms

#### Website blocking (workspace-inverted)

MVP approach: maintain a large curated blocklist (~500 distraction domains) in the hosts file. Allowed sites from the workspace are excluded from this list. Block page server on 127.0.0.1:9512.

**macOS:**
- Modify /etc/hosts, redirect blocked domains to 127.0.0.1
- Flush DNS: dscacheutil -flushcache && killall -HUP mDNSResponder
- Protect hosts file: chflags schg /etc/hosts
- Post-MVP: pf firewall for true whitelist enforcement

**Windows:**
- Modify C:\Windows\System32\drivers\etc\hosts
- Flush DNS: ipconfig /flushdns
- Protect: Set ACL to deny write access
- Post-MVP: WFP API for true whitelist enforcement

#### Application blocking (workspace-inverted)

On seal: enumerate running processes, kill anything not in allowed_apps or system allowlist. Monitor continuously at 500ms intervals.

**macOS:** syscall.Kill with SIGKILL, kqueue or polling for new launches.
**Windows:** CreateToolhelp32Snapshot + TerminateProcess, WMI for real-time events.

**CRITICAL: System process allowlist.** Never kill: Finder, SystemUIServer, Dock, WindowServer, loginwindow, kernel_task (macOS) or explorer.exe, csrss.exe, winlogon.exe, dwm.exe, svchost.exe (Windows). silo itself is always exempt.

#### Self-protection

**MVP:** Single process. Lock is annoying to break (200+ random chars) but a determined user could kill silo via Task Manager.

**v1.1:** Dual-process watchdog. Main process + guardian daemon. Each monitors the other.
- macOS: LaunchDaemon at /Library/LaunchDaemons/com.silo.guardian.plist
- Windows: Windows Service via golang.org/x/sys/windows/svc

---

## MVP scope

### Core features (must ship)

1. **Workspace management**
   - Create, edit, delete named workspaces
   - Each workspace: allowed_apps, allowed_sites, obsidian_vault, obsidian_note
   - Pre-built templates (coding, studying, writing, research, leetcode, nuclear)
   - Import/export as JSON

2. **Workspace-inverted website blocking**
   - Hosts file with curated distraction list, excluding workspace's allowed sites
   - Block page on 127.0.0.1:9512 with workspace-aware messaging
   - Wildcard support: localhost:*, *.react.dev

3. **Workspace-inverted app blocking**
   - Kill non-allowed processes on seal, monitor for new launches (500ms poll)
   - System process allowlist (never kill)

4. **Quick exceptions (escape valve)**
   - [x] during seal → type domain/app → type "i need this" → temporary allowance
   - Logged, session-scoped, permanence prompt post-session

5. **Session lifecycle**
   - IDLE → CONFIGURING → SEALED → COMPLETED
   - Activation ramp: "What are you working on?" + "First tiny step?"
   - 3-second countdown after confirmation, no cancel
   - Commit message prompt on completion

6. **Lock system**
   - random-text: type N chars (escalates 200 → 400 → 600 per attempt)
   - timer: no unlock until expiry
   - reboot: must restart machine

7. **Obsidian integration (MVP)**
   - Workspace stores optional vault name + note path
   - On seal: auto-open note via obsidian:// URI
   - On complete: append session log to daily note (configurable)

8. **Statistics**
   - Session log with task, duration, workspace, commit message, exceptions
   - Daily/weekly focus time, streak counter
   - JSON export

9. **System integration**
   - Auto-enable DND (macOS Focus / Windows Focus Assist) on seal

### Deferred to post-MVP

- Training mode (keyboard shortcut drills — see full spec below)
- Sprint mode (adaptive Pomodoro with breaks)
- Scheduled seals (recurring weekly)
- Watchdog/guardian process
- CLI companion
- Ambient sounds
- Browser extension (URL-path-level filtering)
- Firewall-level blocking (true whitelist via pf/WFP)
- Obsidian plugin

---

## UI specification

### Design philosophy

TUI rendered in a desktop window. Think lazygit, htop, k9s.

**Principles:**
- Monospace everything (JetBrains Mono, Fira Code, or system mono)
- Box-drawing characters for borders: ┌ ─ ┐ │ └ ─ ┘ ├ ┤
- Dark terminal palette (#0d1117 bg, #c9d1d9 text)
- Minimal semantic color (green=allowed/active, blue=interactive, yellow=warning, red=blocked/destructive, dim gray=borders/secondary)
- Keyboard-first with shortcuts in footer
- No animations, no transitions, instant state changes
- Text characters for indicators: ▸ ○ ● ✕ █ ✓

### Color palette

    --bg-primary: #0d1117
    --bg-secondary: #161b22
    --bg-tertiary: #21262d
    --text-primary: #f0f6fc
    --text-secondary: #c9d1d9
    --text-dim: #484f58
    --accent-green: #3fb950
    --accent-blue: #58a6ff
    --accent-yellow: #d29922
    --accent-red: #f85149
    --accent-purple: #bc8cff
    --border: #30363d

### Screen 1: Main dashboard

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo  v0.1.0                                    HH:MM AM │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  WORKSPACES                            status                │
    │                                                              │
    │  ▸ react-project   5 sites · 3 apps    ○ idle               │
    │    ml-study        4 sites · 3 apps    ○ idle               │
    │    writing         3 sites · 2 apps    ○ idle               │
    │    leetcode        4 sites · 2 apps    ○ idle               │
    │                                                              │
    │  ────────────────────────────────────────────────────────     │
    │                                                              │
    │  TODAY          ████████░░░░  2h 14m focused                 │
    │  THIS WEEK      ██████████░░  11h 02m focused                │
    │  STREAK         7 days                                       │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [enter] seal [e] edit [n] new [t] templates [s] stats [q] quit│
    └──────────────────────────────────────────────────────────────┘

- j/k or arrow keys to navigate workspace list
- Summary shows ALLOWED counts (not blocked)
- [t] opens template picker

### Screen 2: Seal configuration

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo › react-project › seal                                │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  WORKSPACE SUMMARY                                           │
    │  apps:  VS Code, Terminal, Chrome                            │
    │  sites: localhost:*, react.dev, claude.ai, github.com +1     │
    │  notes: CS/projects/react-project-log                        │
    │                                                              │
    │  DURATION                                                    │
    │  [ 90 min ]  ◄ ►  or type a number                          │
    │                                                              │
    │  LOCK TYPE                                                   │
    │  ▸ random-text   type 200 random chars to unlock             │
    │    reboot        must restart machine to unlock              │
    │    timer         cannot unlock until timer expires            │
    │                                                              │
    │  WHAT ARE YOU WORKING ON?                                    │
    │  > _                                                         │
    │                                                              │
    │  FIRST TINY STEP?                                            │
    │  > _                                                         │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [enter] SEAL — no going back                     [esc] cancel│
    └──────────────────────────────────────────────────────────────┘

- Workspace summary at top for review before committing
- "notes:" shows Obsidian note that will auto-open
- Both text inputs required
- Red "SEAL" text, 3-second countdown after confirm

### Screen 3: Active seal

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo › LOCKED                                              │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │                     01:12:34                                  │
    │                   remaining                                   │
    │                                                              │
    │  workspace   react-project                                   │
    │  task        build auth flow with OAuth2                     │
    │  lock        random-text (200 chars)                         │
    │  elapsed     17:26                                           │
    │  exceptions  1 added this session                            │
    │                                                              │
    │  ──────────────────────────────────────────────────────────  │
    │                                                              │
    │  blocked just now:                                           │
    │    ✕ twitter.com         3 sec ago                           │
    │    ✕ Spotify.app         1 min ago                           │
    │    ✕ Discord.app         4 min ago                           │
    │                                                              │
    │  "that's not what you're doing right now."                   │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [x] quick exception  [u] unlock (200 chars)     lock active │
    └──────────────────────────────────────────────────────────────┘

### Screen 4: Workspace editor

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo › react-project › edit                                │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  ALLOWED APPS (3)                                            │
    │    ✓ VS Code      ✓ Terminal      ✓ Chrome                  │
    │                                                              │
    │  ALLOWED SITES (5)                                           │
    │    ✓ localhost:*          ✓ react.dev                        │
    │    ✓ claude.ai            ✓ github.com                      │
    │    ✓ stackoverflow.com                                       │
    │                                                              │
    │  OBSIDIAN                                                    │
    │    vault: CS                                                 │
    │    note:  projects/react-project-log                         │
    │                                                              │
    │  ──────────────────────────────────────────────────────────  │
    │  everything not listed above is BLOCKED during seal          │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [a] add  [d] delete  [o] obsidian  [tab] section  [esc] back│
    └──────────────────────────────────────────────────────────────┘

- Green ✓ prefix for allowed items
- "everything not listed above is BLOCKED" reminder
- [o] configures Obsidian vault + note path

### Screen 5: Stats view

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo › stats                                               │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  FOCUS SUMMARY                                               │
    │  today          2h 14m     ████████░░░░                      │
    │  this week      11h 02m    ██████████░░                      │
    │  streak         7 days     ●●●●●●●○○○                        │
    │                                                              │
    │  ──────────────────────────────────────────────────────────  │
    │                                                              │
    │  RECENT SESSIONS                                             │
    │  today 10:30    90m  react-project  build auth flow          │
    │  today 08:00    45m  ml-study       ISLR chapter 10          │
    │  yesterday      120m react-project  refactor state mgmt     │
    │  apr 13         60m  writing        AI ethics presentation   │
    │  apr 12         90m  leetcode       graph problems           │
    │                                                              │
    │  total sessions: 34    total focus: 42h 15m                  │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [j/k] scroll  [enter] details  [x] export json  [esc] back  │
    └──────────────────────────────────────────────────────────────┘

### Screen 6: Unlock attempt

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo › LOCKED › breach attempt #1                          │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  type the following text exactly to unlock:                  │
    │                                                              │
    │  xK9mP2vL8nQ4wR7jT1yB6hF3dA0sC5eG8iU2oM9xZ4pN7kW          │
    │  1bJ6tH3rY8qV0lD5fX2cS9aE4gI7uO0mK3nP6wR1jT8yB5          │
    │  hF2dA9sC4eG7iU0oM1xZ6pN3kW8bJ5tH2rY9qV4lD1fX6c          │
    │  S3aE8gI5uO2mK7nP0wR9jT4yB1hF6dA3sC8eG5iU                 │
    │                                                              │
    │  ──────────────────────────────────────────────────────────  │
    │  > _                                                         │
    │  progress: 0/200 characters                                  │
    │  ⚠ attempt #2 will require 400 characters                   │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [esc] cancel attempt (lock stays active)                     │
    └──────────────────────────────────────────────────────────────┘

### Screen 7: Quick exception

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo › LOCKED › quick exception                            │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  need something not in your workspace?                       │
    │                                                              │
    │  type [site] or [app]:                                       │
    │  ▸ site                                                      │
    │    app                                                       │
    │                                                              │
    │  domain or app name:                                         │
    │  > docs.python.org█                                          │
    │                                                              │
    │  to confirm, type "i need this":                             │
    │  > _                                                         │
    │                                                              │
    │  ⚠ this exception lasts for this session only                │
    │  ⚠ it will be logged in your session history                 │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [enter] add exception                        [esc] cancel    │
    └──────────────────────────────────────────────────────────────┘

### Screen 8: Template picker

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo › new workspace › templates                           │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  choose a template:                                          │
    │                                                              │
    │  ▸ coding       VS Code, Terminal, Chrome                    │
    │                  localhost, github, SO, claude, MDN           │
    │                                                              │
    │    studying     Obsidian, Chrome, PDF reader                 │
    │                  youtube, claude (+ add your course sites)   │
    │                                                              │
    │    writing      Obsidian, Chrome                             │
    │                  claude, google scholar                      │
    │                                                              │
    │    research     Chrome, Obsidian, Notes                      │
    │                  google, scholar, arxiv, claude              │
    │                                                              │
    │    leetcode     VS Code, Terminal, Chrome                    │
    │                  leetcode, neetcode, claude, cppreference    │
    │                                                              │
    │    nuclear      (nothing allowed)                            │
    │    blank        start from scratch                           │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [enter] select                               [esc] cancel    │
    └──────────────────────────────────────────────────────────────┘

### Screen 9: Block page (browser)

    ┌──────────────────────────────────────────────────────────────┐
    │                                                              │
    │            ACCESS DENIED                                     │
    │                                                              │
    │            twitter.com is not in your workspace              │
    │            workspace: react-project                          │
    │            remaining: 01:12:34                               │
    │                                                              │
    │            "that's not what you're doing right now."         │
    │                                                              │
    │            task: build auth flow with OAuth2                 │
    │                                                              │
    │            need this site? press [x] in silo                │
    │                                                              │
    └──────────────────────────────────────────────────────────────┘

### Navigation structure

    Main Dashboard
    ├── [enter] Seal Config → Active Seal → Completion
    │                           ├── [x] Quick Exception
    │                           └── [u] Unlock Attempt
    ├── [e] Workspace Editor
    ├── [n] New Workspace → name input → editor
    ├── [t] Template Picker → name input → editor
    ├── [s] Stats View → Session Detail
    └── [q] Quit (blocked during seal)

### Window properties

- Default: 720 × 520px, min 600 × 400px, resizable
- System tray: green ■ (idle), red ■ (sealed)
- Tray menu: status, open silo, quit (disabled during seal)

---

## Data model (SQLite)

    CREATE TABLE workspaces (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL UNIQUE,
        allowed_apps TEXT NOT NULL DEFAULT '[]',
        allowed_sites TEXT NOT NULL DEFAULT '[]',
        obsidian_vault TEXT,
        obsidian_note TEXT,
        template_source TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE sessions (
        id TEXT PRIMARY KEY,
        workspace_id TEXT NOT NULL,
        task_description TEXT NOT NULL,
        first_step TEXT NOT NULL,
        commit_message TEXT,
        lock_type TEXT NOT NULL,
        lock_chars INTEGER,
        duration_planned INTEGER NOT NULL,
        duration_actual INTEGER,
        started_at DATETIME NOT NULL,
        completed_at DATETIME,
        status TEXT NOT NULL DEFAULT 'active',
        breach_attempts INTEGER DEFAULT 0,
        blocked_attempts TEXT DEFAULT '[]',
        quick_exceptions TEXT DEFAULT '[]',
        FOREIGN KEY (workspace_id) REFERENCES workspaces(id)
    );

    CREATE TABLE daily_stats (
        date TEXT PRIMARY KEY,
        total_focus_seconds INTEGER DEFAULT 0,
        session_count INTEGER DEFAULT 0,
        blocked_attempt_count INTEGER DEFAULT 0
    );

    CREATE TABLE settings (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL
    );

    INSERT INTO settings (key, value) VALUES
        ('default_duration', '5400'),
        ('default_lock_type', 'random-text'),
        ('default_lock_chars', '200'),
        ('escalation_step', '200'),
        ('dnd_integration', 'true'),
        ('show_blocked_log', 'true'),
        ('block_page_port', '9512'),
        ('obsidian_daily_note_path', ''),
        ('obsidian_vault_fs_path', '');

---

## File structure

    silo/
    ├── README.md
    ├── SILO_DESIGN.md
    ├── go.mod / go.sum / main.go / wails.json
    │
    ├── app/
    │   ├── app.go
    │   ├── blocker/
    │   │   ├── blocker.go          — blocking engine interface
    │   │   ├── hosts.go            — hosts file manipulation
    │   │   ├── hosts_darwin.go     — macOS paths/commands
    │   │   ├── hosts_windows.go    — Windows paths/commands
    │   │   ├── process.go          — process monitor & killer
    │   │   ├── process_darwin.go
    │   │   ├── process_windows.go
    │   │   ├── allowlist.go        — system process allowlist
    │   │   └── blockpage.go        — block page HTTP server
    │   ├── session/
    │   │   ├── session.go          — session lifecycle
    │   │   ├── timer.go            — countdown with tick events
    │   │   ├── lock.go             — lock generation & validation
    │   │   └── exception.go        — quick exception logic
    │   ├── workspace/
    │   │   ├── workspace.go        — CRUD operations
    │   │   ├── templates.go        — built-in templates
    │   │   └── obsidian.go         — URI launch + daily note append
    │   ├── stats/
    │   │   └── stats.go
    │   ├── store/
    │   │   ├── db.go               — SQLite connection & migrations
    │   │   └── migrations.go
    │   └── platform/
    │       ├── dnd.go / dnd_darwin.go / dnd_windows.go
    │       └── open.go / open_darwin.go / open_windows.go
    │
    ├── frontend/
    │   ├── package.json / tsconfig.json / index.html
    │   └── src/
    │       ├── main.tsx / App.tsx
    │       ├── styles/terminal.css
    │       ├── screens/
    │       │   ├── Dashboard.tsx / SealConfig.tsx / ActiveSeal.tsx
    │       │   ├── WorkspaceEditor.tsx / Stats.tsx / UnlockAttempt.tsx
    │       │   ├── QuickException.tsx / TemplatePicker.tsx
    │       │   └── (post-MVP) TrainingMode.tsx
    │       ├── components/
    │       │   ├── TuiBox / TuiFooter / TuiHeader / TuiList / TuiInput
    │       │   ├── ProgressBar / Timer
    │       │   └── (post-MVP) DrillPrompt
    │       ├── hooks/
    │       │   ├── useKeyboard.ts / useTimer.ts / useNavigation.ts
    │       │   └── (post-MVP) useKeyCapture.ts
    │       └── lib/
    │           ├── wails.ts / format.ts
    │           └── (post-MVP) shortcuts.ts
    │
    └── build/
        ├── darwin/Info.plist
        └── windows/info.json

---

## Implementation order

### Phase 1: Skeleton (day 1)
1. wails init -n silo -t react-ts
2. Terminal CSS, color variables, monospace font
3. TUI component library: TuiBox, TuiHeader, TuiFooter, TuiList, TuiInput
4. Dashboard screen with hardcoded data
5. Keyboard navigation (useKeyboard hook)

### Phase 2: Workspaces (day 2)
1. SQLite schema + migrations
2. Workspace CRUD in Go
3. Templates in templates.go
4. Dashboard wired to real data
5. TemplatePicker screen
6. WorkspaceEditor screen
7. Obsidian vault/note config in editor

### Phase 3: Blocking engine (days 3-5)
1. Hosts file read/write (cross-platform)
2. DNS flush (cross-platform)
3. Curated distraction domain list (~500 domains)
4. Process enumeration + system allowlist
5. Process monitor loop (500ms) — kill non-allowed
6. Block page HTTP server on 127.0.0.1:9512
7. Integration test: seal workspace, verify enforcement
8. Hosts file protection during seal

### Phase 4: Sessions (days 5-7)
1. SealConfig screen with workspace summary
2. Session creation in SQLite
3. Countdown timer (Go goroutine → Wails events)
4. Obsidian auto-open on seal
5. ActiveSeal screen with timer + blocked log
6. QuickException screen + live allowlist update
7. Session completion: commit message + exception permanence prompt
8. Lock generation (crypto/rand, a-zA-Z0-9)
9. UnlockAttempt screen with char-by-char validation
10. Escalating lock (200 → 400 → 600)
11. Obsidian daily note append on complete

### Phase 5: Stats & polish (day 8)
1. Stats queries, streak calculation
2. Stats screen with session history
3. JSON export
4. System tray icon
5. DND integration
6. Edge cases: crash recovery, midnight rollover

### Phase 6: Hardening (day 9+)
1. Cross-platform testing
2. Edge cases: VPN, custom DNS, multiple browsers
3. App icon, metadata
4. Installers: .dmg (macOS), .msi (Windows)

---

## Training mode (planned — post-MVP)

### Concept

Built-in drill mode that teaches keyboard shortcuts for efficient workflow navigation. Eliminates mouse dependency so switching between allowed tools during a sealed session is fast and frictionless.

### Drill categories

**App switching**
- macOS: Cmd+Tab, Cmd+Shift+Tab, Cmd+` (same-app windows)
- Windows: Alt+Tab, Alt+Shift+Tab, Win+Tab

**Browser navigation**
- Ctrl/Cmd+L (address bar), Ctrl/Cmd+T (new tab), Ctrl/Cmd+W (close tab)
- Ctrl+Tab / Ctrl+Shift+Tab (next/prev tab)
- Ctrl/Cmd+Shift+T (reopen closed tab), Ctrl/Cmd+1-9 (tab by position)

**Editor shortcuts (VS Code)**
- Ctrl/Cmd+P (quick open), Ctrl/Cmd+Shift+P (command palette)
- Ctrl/Cmd+` (terminal), Ctrl/Cmd+B (sidebar)
- Ctrl/Cmd+\ (split), Ctrl+1/2/3 (focus group)

**Terminal shortcuts**
- Ctrl+A (line start), Ctrl+E (line end)
- Ctrl+R (reverse search), Ctrl+U/K (clear line)
- Tab (autocomplete)

**Window management**
- macOS: Ctrl+Left/Right (spaces), F3/Ctrl+Up (Mission Control)
- Windows: Win+Left/Right (snap), Win+Up/Down, Win+D (desktop)

**Custom drills** — users define their own shortcut sets, import/export as JSON.

### Training UI

    ┌──────────────────────────────────────────────────────────────┐
    │ ■ silo › training › browser navigation                       │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  DRILL 4/10                                                  │
    │                                                              │
    │  action:  jump to tab #3                                     │
    │                                                              │
    │  press the right keys: _                                     │
    │                                                              │
    │  ──────────────────────────────────────────────────────────  │
    │                                                              │
    │  streak:  ●●●○○○○○○○  3/10                                  │
    │  best:    7/10                                               │
    │  speed:   avg 1.2s per drill                                 │
    │                                                              │
    ├──────────────────────────────────────────────────────────────┤
    │ [h] toggle hints  [r] restart  [tab] next category  [esc] back│
    └──────────────────────────────────────────────────────────────┘

- Platform-aware (detects macOS vs Windows, shows correct keys)
- Hints toggle: on = answer shown, off = recall from memory
- Speed tracking per drill
- 10 drills per round, shuffled
- Wrong: red flash + correct answer shown 1 sec. Right: green flash + advance.
- Best scores persisted in SQLite

### Training data model

    CREATE TABLE training_scores (
        id TEXT PRIMARY KEY,
        category TEXT NOT NULL,
        drills_completed INTEGER,
        drills_correct INTEGER,
        avg_speed_ms INTEGER,
        best_streak INTEGER,
        completed_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE custom_drills (
        id TEXT PRIMARY KEY,
        category_name TEXT NOT NULL,
        drills TEXT NOT NULL DEFAULT '[]'
    );

### Training implementation notes

- Key capture: intercept multi-key combos in React without triggering OS actions
- Platform detection via runtime.GOOS exposed as Wails binding
- Accessible from dashboard via [d] drills shortcut
- Available during idle state only (not during active seal)

---

## CLI interface (post-MVP)

    silo seal --workspace react-project --duration 90m --lock random-text
    silo status
    silo workspaces
    silo stats / stats --week / stats --export
    silo breach
    silo exception docs.python.org
    silo train browser

Backend exposes local Unix socket (macOS) or named pipe (Windows). CLI and Wails frontend call the same Go functions.

---

## Key design decisions

| Decision | Rationale |
|---|---|
| Workspace-first (allow, not block) | Eliminates whack-a-mole. 5 allowed sites = zero distractions vs 50 blocked = infinite remaining. |
| Quick exceptions with friction | Solves over-blocking without breaking the model. "i need this" stops impulse, not need. |
| Templates | Reduces setup friction. Lowers executive function cost of getting started. |
| Activation ramp (task + first step) | ADHD research: activation barrier is the real enemy. |
| Obsidian auto-open on seal | Environmental design. Notes are ready when you are. |
| Obsidian daily note logging | Progress trail in YOUR system, not locked in silo's DB. |
| Escalating lock | Each failed attempt = more friction. Converts impulse into barrier. |
| "Not in your workspace" framing | Non-judgmental redirect. Reflects user's own stated intention. |
| Training mode | Reduces switching cost WITHIN allowed tools. 2 sec keyboard switch vs 10 sec mouse. |
| Commit message on completion | Dopamine reward via visible progress. Reviewable log. |
| No cloud, no accounts | Privacy-first. No "check settings online" distraction vector. |
| Terminal aesthetic | Signals tool, not app. Reduces visual noise. Native to target user's workflow. |

---

## Notes for Claude Code

1. Start with UI skeleton. Terminal CSS + TUI components first.
2. Workspace model inverts blocking: block everything EXCEPT allowed. For MVP, use curated distraction list in hosts file, exclude allowed sites. Post-MVP, firewall for true whitelist.
3. Use Go build tags (//go:build darwin/windows) for platform code. No runtime.GOOS inline.
4. Hosts file needs elevated privileges. Configure Wails to request elevation.
5. wails generate module after Go changes to update TypeScript bindings.
6. Test blocking engine independently with Go unit tests.
7. Session state must survive crashes. Write to SQLite on seal, restore on launch.
8. Block page server must start BEFORE hosts file modification.
9. Quick exceptions: update hosts file in real-time, flush DNS, persist to session record.
10. Obsidian = URI launch + file append. Don't over-engineer.
11. Lock text: crypto/rand, charset a-zA-Z0-9 only. Store in session record.
12. Frontend: no animations, no transitions, instant screen swaps.
13. System process allowlist is CRITICAL. Never kill Finder/explorer/etc. Err on the side of allowing too many system processes.