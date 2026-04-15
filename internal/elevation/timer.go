// Package elevation handles admin-permission acquisition: displays a banner,
// prompts the user to confirm, and tracks the elevation window duration so
// the runner can warn as it nears expiry.
package elevation

import "time"

// Timer tracks how long an elevation window has been open.
// A Timer constructed with a zero duration reports Remaining()==0 and
// IsNearExpiry()==false regardless of threshold — used when the recipe
// did not configure an advisory duration.
type Timer struct {
	start    time.Time
	duration time.Duration
}

// NewTimer returns a Timer started now.
func NewTimer(duration time.Duration) *Timer {
	return &Timer{start: time.Now(), duration: duration}
}

// Remaining returns the time left in the elevation window, or 0 if
// the timer has no configured duration.
func (t *Timer) Remaining() time.Duration {
	if t.duration == 0 {
		return 0
	}
	r := t.duration - time.Since(t.start)
	if r < 0 {
		return 0
	}
	return r
}

// IsNearExpiry reports whether the remaining window is less than threshold.
// Returns false if the timer has no configured duration.
func (t *Timer) IsNearExpiry(threshold time.Duration) bool {
	if t.duration == 0 {
		return false
	}
	return t.Remaining() < threshold
}
