package elevation_test

import (
	"testing"
	"time"

	"gearup/internal/elevation"
)

func TestTimer_ZeroDurationMeansNoExpiry(t *testing.T) {
	tm := elevation.NewTimer(0)
	if tm.Remaining() != 0 {
		t.Errorf("Remaining = %v, want 0 (no duration set)", tm.Remaining())
	}
	if tm.IsNearExpiry(30 * time.Second) {
		t.Error("IsNearExpiry = true, want false (no duration set)")
	}
}

func TestTimer_RemainingCountsDown(t *testing.T) {
	tm := elevation.NewTimer(500 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	got := tm.Remaining()
	if got <= 0 || got >= 500*time.Millisecond {
		t.Errorf("Remaining = %v, want between 0 and 500ms", got)
	}
}

func TestTimer_IsNearExpiryTrueWhenWithinThreshold(t *testing.T) {
	tm := elevation.NewTimer(50 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	if !tm.IsNearExpiry(30 * time.Millisecond) {
		t.Errorf("IsNearExpiry = false with remaining %v, want true", tm.Remaining())
	}
}

func TestTimer_IsNearExpiryFalseWhenOutsideThreshold(t *testing.T) {
	tm := elevation.NewTimer(1 * time.Second)
	if tm.IsNearExpiry(30 * time.Millisecond) {
		t.Errorf("IsNearExpiry = true with remaining %v, want false", tm.Remaining())
	}
}
