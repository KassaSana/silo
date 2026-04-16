package platform

/*
 * dnd.go — Cross-platform "Do Not Disturb" toggle.
 *
 * CONCEPT: When silo seals, we tell the OS to suppress notifications so
 * focus isn't broken by a Slack ping. When the session ends, we restore
 * notifications.
 *
 * The concrete implementation lives in dnd_darwin.go / dnd_windows.go.
 * This file defines the shared interface: EnableDND() and DisableDND().
 *
 * WHY build tags instead of runtime.GOOS? Compile-time platform selection.
 * The darwin build doesn't even TRY to compile the Windows registry code,
 * and vice versa. Fewer cross-platform surprises.
 *
 * Both implementations are best-effort — a failure to toggle DND should
 * never break the seal. Callers should log and continue.
 */

// Empty — see dnd_darwin.go / dnd_windows.go for the real implementations.
