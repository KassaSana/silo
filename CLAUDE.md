# silo — project instructions for Claude

A local-first desktop focus app (Go + React + Wails + SQLite).
Cross-platform: macOS + Windows.

## Stack context
- **Backend:** Go (modernc.org/sqlite, no cgo). Platform-split via build tags (`_darwin.go`, `_windows.go`).
- **Frontend:** React + TypeScript. No React Router — state-machine routing in [App.tsx](frontend/src/App.tsx).
- **Bridge:** Wails v2 — Go methods on `App` struct become JS bindings. Events via `runtime.EventsEmit`.
- **Persistence:** `~/.silo/silo.db` (SQLite WAL mode).
- **OS surface:** hosts file, DNS flush, process enumeration/kill, file protection flags (chflags/icacls). All require elevated privileges when sealed.

---

## Security defaults (always apply before declaring code "done")

Silo runs with elevated privileges during a seal. That makes input-handling mistakes expensive — a mistyped domain could corrupt `/etc/hosts`, a bad process name could match `kernel_task`. Default to paranoia.

### Input validation & trust boundaries
- **Treat every user input as untrusted**, even in a local app. User-typed workspace names, domains, app names all flow into system-modifying code paths.
- **Validate at the boundary, not deep in the stack.** Reject bad input where it enters (Go handler) — don't assume downstream code will catch it.
- **Never concatenate user input into shell commands.** Use `exec.Command("ps", "-eo", "pid,comm")` with separate args, never `exec.Command("sh", "-c", "ps "+userInput)`.
- **Never concatenate into SQL.** Parameterized queries only (`?` placeholders). The codebase already does this — don't break the pattern.
- **Sanitize before writing to `/etc/hosts`:** strip newlines, tabs, control chars. A domain with `\n127.0.0.1 evil.com` would inject an arbitrary hosts entry.
- **Domain validation:** match against a strict regex (`^[a-zA-Z0-9.-]+(:\d+)?$` with wildcard allowance), not "anything that isn't whitespace."

### Privilege & blast radius
- **Principle of least privilege.** Only request elevation for the narrow operation that needs it (hosts write, chflags, icacls). Never run the whole app as root.
- **System process allowlist is sacred.** Before adding to [allowlist.go](app/blocker/allowlist.go), verify the name isn't attacker-controllable. Never match by PID alone — PIDs recycle.
- **Fail closed on blocking.** If hosts write fails mid-seal, restore from backup before returning. A half-applied block is worse than no block.
- **Backup before modify.** Every destructive system op (hosts write, chflags) should leave a recoverable state. Already in [hosts.go](app/blocker/hosts.go) — preserve this.

### Secrets & data
- **No secrets in source or frontend bundle.** Silo has none today; if cloud sync is ever added, keys live in OS keychain (macOS Keychain / Windows Credential Manager), not `.env`.
- **SQLite file permissions:** `~/.silo/silo.db` should be `0600` (user-only). Check when creating.
- **No telemetry without explicit opt-in.** Even anonymized.

### Dependency hygiene
- **Justify every new dep.** When adding to `go.mod` or `package.json`, note why it was chosen over stdlib and what its trust source is (popular? audited? actively maintained?).
- **Prefer stdlib.** Go stdlib is audited; random GitHub packages are not.
- **Lock versions.** `go.sum` and `package-lock.json` stay committed.

### Error handling & logging
- **Never swallow errors from system ops.** `os.WriteFile` returning an error on `/etc/hosts` write is critical — surface it to the UI, don't log-and-continue.
- **Log what got killed, blocked, or modified.** User needs an audit trail: "silo killed these processes at HH:MM:SS." Already in [process.go](app/blocker/process.go) — keep this.
- **Don't log user task descriptions to disk unencrypted** if they might contain sensitive context (PII, passwords). Current storage is local-only, so this is low risk — revisit if sync is added.

---

## Quality defaults

### Testing expectations
- **Business logic gets unit tests** — lock escalation math, streak calculation, template instantiation, hosts-file merge logic.
- **System ops get integration tests** — hosts read/write, process listing. Mark them `//go:build integration` so they don't run in CI by default.
- **UI gets smoke tests, not exhaustive coverage** — keyboard flow through each screen.
- Flag missing tests for new business logic before declaring a feature done.

### Before "done"
- Code compiles cleanly (`go build ./...`, `tsc --noEmit`).
- No `TODO:` or `FIXME:` left in new code without a corresponding checklist entry.
- New deps justified in the commit message.
- New user input paths: verify validation exists at the boundary.
- New system ops: verify backup/restore path exists.

### Commenting discipline
- Comment the **why**, not the **what**. `// kill with SIGKILL because SIGTERM lets Chrome respawn` > `// send SIGKILL`.
- Platform-specific quirks always get a comment (why chflags? because `chmod` is insufficient on macOS for this).

---

## Tutor mode

The user is a beginner in web dev, stronger in systems. When explaining:
- Frame web/frontend concepts via analogy to systems concepts they already know.
- Ask "what do you think happens if..." before explaining.
- Smallest hint first, then escalate.
- Explain *why*, not just *that*.
