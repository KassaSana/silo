package session

import (
	"strings"
	"testing"
)

// EscalateChars is the core of the "pain escalates with each failed unlock"
// design. Get this wrong and either users breeze past the penalty, or the
// app becomes unusable after one mistake.

func TestEscalateChars_FirstAttemptIsBase(t *testing.T) {
	// Attempt 1 should yield exactly the base — no escalation yet.
	if got := EscalateChars(1, 200, 200); got != 200 {
		t.Errorf("attempt 1 = %d, want 200", got)
	}
}

func TestEscalateChars_LinearGrowth(t *testing.T) {
	cases := []struct {
		attempt, want int
	}{
		{1, 200},
		{2, 400},
		{3, 600},
		{4, 800},
		{10, 2000},
	}
	for _, c := range cases {
		if got := EscalateChars(c.attempt, 200, 200); got != c.want {
			t.Errorf("EscalateChars(%d, 200, 200) = %d, want %d", c.attempt, got, c.want)
		}
	}
}

func TestEscalateChars_CustomStep(t *testing.T) {
	// A user could configure a milder step (say 100) — verify the math honours it.
	if got := EscalateChars(3, 50, 100); got != 250 {
		t.Errorf("got %d, want 250", got)
	}
}

// GenerateLockText uses crypto/rand. We can't test randomness directly,
// but we CAN verify length + charset constraints.

func TestGenerateLockText_Length(t *testing.T) {
	for _, n := range []int{0, 1, 10, 200, 1000} {
		out, err := GenerateLockText(n)
		if err != nil {
			t.Fatalf("GenerateLockText(%d) errored: %v", n, err)
		}
		if len(out) != n {
			t.Errorf("length mismatch: got %d, want %d", len(out), n)
		}
	}
}

func TestGenerateLockText_CharsetOnly(t *testing.T) {
	// Design spec says a-zA-Z0-9 only. Any other char is a regression
	// (e.g. special chars would be painful to type on a phone keyboard).
	out, err := GenerateLockText(500)
	if err != nil {
		t.Fatalf("GenerateLockText errored: %v", err)
	}
	for i, c := range out {
		if !strings.ContainsRune(charset, c) {
			t.Errorf("char at %d is %q, outside charset", i, c)
		}
	}
}

// ValidateLockText is the per-char comparator that the UnlockAttempt
// screen uses to show progress. The key property: we short-circuit on
// the first mismatch (don't count characters past the error).

func TestValidateLockText_ExactMatch(t *testing.T) {
	ok, correct := ValidateLockText("abc123", "abc123")
	if !ok {
		t.Error("exact match should return true")
	}
	if correct != 6 {
		t.Errorf("correct = %d, want 6", correct)
	}
}

func TestValidateLockText_PartialPrefix(t *testing.T) {
	ok, correct := ValidateLockText("abc", "abc123")
	if ok {
		t.Error("partial input should return false for match")
	}
	if correct != 3 {
		t.Errorf("correct = %d, want 3", correct)
	}
}

func TestValidateLockText_StopsAtFirstMismatch(t *testing.T) {
	// Input "abXdef" vs lock "abcdef" — must stop at index 2, not count "def".
	_, correct := ValidateLockText("abXdef", "abcdef")
	if correct != 2 {
		t.Errorf("correct = %d, want 2 (should stop at first mismatch)", correct)
	}
}

func TestValidateLockText_EmptyInput(t *testing.T) {
	ok, correct := ValidateLockText("", "abc")
	if ok || correct != 0 {
		t.Errorf("empty input: got (%v, %d), want (false, 0)", ok, correct)
	}
}
