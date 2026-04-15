package elevation_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gearup/internal/config"
	"gearup/internal/elevation"
)

func TestAcquire_PrintsMessageAndCallsPrompter(t *testing.T) {
	fake := elevation.NewFakePrompter()
	fake.Result = true

	var buf bytes.Buffer
	cfg := &config.Elevation{
		Message:  "Please elevate admin, then continue.",
		Duration: 30 * time.Second,
	}
	tm, err := elevation.Acquire(context.Background(), cfg, fake, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tm == nil {
		t.Fatal("Timer is nil")
	}
	if !strings.Contains(buf.String(), "Please elevate admin") {
		t.Errorf("banner output missing message: %q", buf.String())
	}
	if fake.Calls() != 1 {
		t.Errorf("prompter calls = %d, want 1", fake.Calls())
	}
}

func TestAcquire_UserAborts_ReturnsError(t *testing.T) {
	fake := elevation.NewFakePrompter()
	fake.Result = false

	cfg := &config.Elevation{Message: "elevate now"}
	_, err := elevation.Acquire(context.Background(), cfg, fake, &bytes.Buffer{})
	if err == nil {
		t.Fatal("want error when user aborts, got nil")
	}
}

func TestAcquire_PrompterError_Propagates(t *testing.T) {
	fake := elevation.NewFakePrompter()
	fake.Err = errors.New("prompter boom")

	cfg := &config.Elevation{Message: "elevate now"}
	_, err := elevation.Acquire(context.Background(), cfg, fake, &bytes.Buffer{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "prompter boom") {
		t.Errorf("error = %v, want wrapping prompter error", err)
	}
}

func TestAcquire_EmptyMessage_ReturnsError(t *testing.T) {
	fake := elevation.NewFakePrompter()
	cfg := &config.Elevation{}
	_, err := elevation.Acquire(context.Background(), cfg, fake, &bytes.Buffer{})
	if err == nil {
		t.Error("want error for empty message, got nil")
	}
}
