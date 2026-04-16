package session

/*
 * timer.go — Countdown timer that emits tick events to the frontend.
 *
 * CONCEPT: The timer runs as a Go goroutine. Every second, it sends
 * a "tick" event to the React frontend via Wails runtime events.
 * The frontend listens for these events and updates the UI.
 *
 * WHY Go goroutine instead of JavaScript setInterval?
 * Because the timer is authoritative — it determines when the session
 * ends. If the frontend timer drifts (tab throttling, etc.), the Go
 * timer is still accurate. The frontend is just a display.
 *
 * Wails events are one-way messages from Go to JS (or JS to Go).
 * Think of them like WebSocket messages, but built into the framework.
 */

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TimerState is sent to the frontend on every tick.
type TimerState struct {
	Remaining int    `json:"remaining"` // seconds remaining
	Elapsed   int    `json:"elapsed"`   // seconds elapsed
	Formatted string `json:"formatted"` // "01:23:45" display string
	Done      bool   `json:"done"`      // true when timer expires
}

// Timer manages a countdown.
type Timer struct {
	ctx      context.Context // Wails app context (needed for runtime.EventsEmit)
	duration int             // total seconds
	elapsed  int
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewTimer creates a timer for the given duration in seconds.
func NewTimer(ctx context.Context, durationSeconds int) *Timer {
	return &Timer{
		ctx:      ctx,
		duration: durationSeconds,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the countdown. Emits "timer:tick" events every second.
func (t *Timer) Start() {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// Emit initial state
		t.emit()

		for {
			select {
			case <-ticker.C:
				t.elapsed++
				t.emit()

				if t.elapsed >= t.duration {
					// Timer expired — session complete
					runtime.EventsEmit(t.ctx, "timer:done")
					return
				}
			case <-t.stopCh:
				return
			}
		}
	}()
}

// Stop halts the timer early (for unlock).
func (t *Timer) Stop() {
	close(t.stopCh)
	t.wg.Wait()
}

// Elapsed returns the current elapsed seconds.
func (t *Timer) Elapsed() int {
	return t.elapsed
}

// emit sends the current timer state to the frontend.
func (t *Timer) emit() {
	remaining := t.duration - t.elapsed
	if remaining < 0 {
		remaining = 0
	}

	state := TimerState{
		Remaining: remaining,
		Elapsed:   t.elapsed,
		Formatted: formatDuration(remaining),
		Done:      remaining <= 0,
	}

	runtime.EventsEmit(t.ctx, "timer:tick", state)
}

// formatDuration converts seconds to "HH:MM:SS" display string.
func formatDuration(totalSeconds int) string {
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
