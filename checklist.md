# Silo Build Checklist

## Completed

- [x] **Phase 1: Skeleton** — Wails init, terminal CSS, TUI components, Dashboard, keyboard nav (see `frontend/src/`)
- [x] **Phase 2: Workspaces** — SQLite schema, CRUD, templates, WorkspaceEditor, Obsidian config (see `app/workspace/`, `app/store/`)
- [x] **Phase 3: Blocking Engine** — hosts file r/w, DNS flush, ~500 distraction domains, process monitor/kill, block page, hosts protection (see `app/blocker/`)
- [x] **Phase 4: Sessions** — SealConfig, timer, ActiveSeal, QuickException, locks, UnlockAttempt, Obsidian auto-open + daily note append (see `app/session/`, `app/workspace/obsidian.go`)
- [x] **Phase 5: Stats & Polish** — streak calc, stats screen, JSON export, DND integration, crash recovery, close-to-background tray (see `app/stats/`, `app/platform/`)
- [x] **Phase 6.0: Security boundary** — input validation (`app/validate/`), SQLite 0600 perms, fail-closed hosts restore, 55 unit tests

---

## Phase 6: Hardening (blocks v0.1 release)

- [ ] 6.1 Cross-platform testing — Windows hosts/icacls/taskkill paths untested on real hardware
- [ ] 6.2 Edge cases — VPN interaction, custom DNS servers, browser DoH bypass (document stance or detect)
- [ ] 6.3 App icon + bundle metadata — `wails.json` missing version/copyright; `build/darwin/Info.plist` + `build/windows/info.json` still template placeholders
- [ ] 6.4 Installers — `.dmg` for macOS, `.msi` via NSIS for Windows (`build/windows/installer/project.nsi` is default scaffold)

---

## Phase 7: Flow State Mode

> Keyboard drill engine. Trains muscle memory for fast tool-switching during sealed sessions. Available from dashboard via `[f]`.

- [ ] 7.1 Flow mode screen (`frontend/src/screens/FlowMode.tsx`) + `[f]` hotkey on dashboard
- [ ] 7.2 `useKeyCapture` hook — intercept multi-key combos without triggering OS actions
- [ ] 7.3 Platform detection — expose `runtime.GOOS` via Wails binding, auto-show Mac vs Windows drills
- [ ] 7.4 Drill engine — 10 drills per round, shuffled, score + speed tracking
- [ ] 7.5 SQLite tables: `training_scores` + `custom_drills`
- [ ] 7.6 Built-in category sets loaded from `drills.md` (macOS, Windows, Chrome, VS Code, terminal, silo)
- [ ] 7.7 Visual feedback — green flash correct, red + answer on wrong, 1s pause then advance
- [ ] 7.8 Speed analytics — running avg, best streak, per-category heatmap
- [ ] 7.9 Hints toggle — `[h]` shows the answer before you try (training wheels)
- [ ] 7.10 Custom drill import/export — user-authored sets for Figma, Photoshop, etc.

---

## Phase 8: Integrations

> Pick 1–2 at a time. Integration sprawl is its own distraction.

- [ ] 8.1 **Obsidian deeper** — parse `- [ ] task` from linked note, show as checklist on active-seal screen. Auto-check on completion. File-watching for live updates.
- [ ] 8.2 **Calendar + scheduled seals** — point at local `.ics` file. Events titled `Deep Work: *` trigger pre-seal prompt 5 min before. Recurring weekly slots ("every weekday 9-11am: auto-seal").
- [ ] 8.3 **Git integration** — on session complete, run `git log --since=started_at`, save commit list alongside commit message. Visible progress trail.
- [ ] 8.4 **Phone bridge** — silo exposes local HTTP endpoint `POST /seal` for Apple Shortcuts / Tasker. Users wire their own "when silo seals, enable phone DND." Silo signals, doesn't own the phone.

---

## Phase 9: ADHD-specific mechanics

> These lean into the ADHD brain's actual reward/attention patterns. Small features, outsized impact.

- [ ] 9.1 **Drift check-ins** — every N min during seal, one-line prompt: "what are you doing right now?" Typed responses become a session journal.
- [ ] 9.2 **Escalating exit friction** — 1st exception: type phrase. 2nd: solve arithmetic. 3rd: 10s voice memo. Each step raises the activation cost of impulse-breaking.
- [ ] 9.3 **Post-session retrospective** — after commit message: "did you actually work on this? (y/n + note)." Trains self-awareness, feeds stats accuracy.
- [ ] 9.4 **Variable reward pings** — random encouragement during seal (streak flame, good-job line). ADHD brains respond to variable reinforcement > fixed. Muted by default.
- [ ] 9.5 **Cold start ramp** — optional 5-min warm-up: only workspace note + editor unlocked (no browser). Reduces "open browser first out of habit."
- [ ] 9.6 **Idle detection** — keyboard + mouse idle 5 min → "are you still here?" Pause timer on "step away." Stops inflated focus hours from AFK.

---

## Phase 10: Platform hardening

- [ ] 10.1 **Watchdog** — `com.silo.guardian.plist` (macOS) / Windows Service. Two-process pact: killing one isn't enough.
- [ ] 10.2 **Firewall** — pf (macOS) / WFP (Windows) true whitelist. Eliminates DoH bypass entirely.
- [ ] 10.3 **Browser extension** — URL-path-level filtering + tab broom (close non-allowed tabs on seal). Domain-only blocking is the current limit.
- [ ] 10.4 **CLI** — `silo seal --workspace X --duration 90m`. Local Unix-socket / named-pipe to the running Go backend.
- [ ] 10.5 **Sprint mode** — adaptive Pomodoro: 25/5, 50/10, 90/15. Each sub-block is a mini-seal; outer lock governs total.
- [ ] 10.6 **Full status-bar tray** — macOS `NSStatusItem` + Windows `Shell_NotifyIcon` with state-aware icon. Blocked on Wails v3 (see 5.4 for current MVP).

---

## Someday / v2 vision

> Ideas worth remembering but not worth building yet. Revisit after v0.1 ships and real users give feedback.

- **Body doubling** — peer pairing over local discovery + ephemeral relay. Two users see each other's workspace + remaining time. Social pressure mechanic. (This is a product, not a feature — needs its own design pass.)
- **Music auto-pilot** — play Spotify/Apple Music URI on seal, pause on unseal. `osascript` / `mpris`. Low value, high "every focus app does this" energy.
- **Ambient sounds** — white/brown noise. Ships as WAV in `assets/`. Same "every app does this" concern.
- **Obsidian reverse plugin** — companion Obsidian plugin to seal from inside a note. Separate TypeScript project, separate repo.
- **Slack/Discord presence** — webhook on seal/unseal. Ironic for an anti-distraction tool. Reconsider only if users ask.

---

## Item counts

| Phase | Items | Status |
|---|---|---|
| 1–5 + 6.0 | 30 | done |
| 6 (hardening) | 4 | **blocks v0.1** |
| 7 (flow drills) | 10 | post-MVP |
| 8 (integrations) | 4 | post-MVP |
| 9 (ADHD mechanics) | 6 | post-MVP |
| 10 (platform) | 6 | post-MVP |
| Someday | 5 | parked |
