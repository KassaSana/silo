package session

/*
 * lock.go — Lock generation and validation.
 *
 * CONCEPT: The lock is what makes silo's seal meaningful. Three types:
 *
 * 1. random-text: Generate N random characters. User must type them
 *    ALL correctly to unlock. Escalates: 200 → 400 → 600 per attempt.
 *    Uses crypto/rand (cryptographically secure random) because we
 *    don't want the string to be predictable.
 *
 * 2. timer: No unlock possible until the timer expires. Period.
 *
 * 3. reboot: Must physically restart the machine. silo detects on
 *    next launch that the session was interrupted and marks it.
 *
 * The design spec says the random text uses charset a-zA-Z0-9 only.
 * No special characters — it's hard enough without shift-key combos.
 */

import (
	"crypto/rand"
	"math/big"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// LockType defines the available lock mechanisms.
type LockType string

const (
	LockRandomText LockType = "random-text"
	LockTimer      LockType = "timer"
	LockReboot     LockType = "reboot"
)

// GenerateLockText creates a random string of the given length.
// Uses crypto/rand for unpredictable output.
func GenerateLockText(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		// crypto/rand.Int returns a random number in [0, max)
		// We use it to pick a random index into our charset
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result), nil
}

// ValidateLockText checks if the user's input matches the lock text.
// Returns (matches, charsCorrect).
func ValidateLockText(input, lockText string) (bool, int) {
	correct := 0
	for i := 0; i < len(input) && i < len(lockText); i++ {
		if input[i] == lockText[i] {
			correct++
		} else {
			break // Stop at first wrong character
		}
	}
	return input == lockText, correct
}

// EscalateChars returns the number of chars for the given attempt number.
// Attempt 1: 200, Attempt 2: 400, Attempt 3: 600, etc.
func EscalateChars(attempt int, baseChars, step int) int {
	return baseChars + (attempt-1)*step
}
