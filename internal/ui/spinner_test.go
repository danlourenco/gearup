package ui_test

import (
	"bytes"
	"testing"
	"time"

	"gearup/internal/ui"
)

func TestStepPrinter_CheckSkip(t *testing.T) {
	var buf bytes.Buffer
	p := ui.NewStepPrinter(&buf)
	p.StartCheck(1, 12, "Git")
	p.FinishSkip(1, 12, "Git")
	if buf.Len() == 0 {
		t.Error("expected output, got empty")
	}
}

func TestStepPrinter_CheckInstall(t *testing.T) {
	var buf bytes.Buffer
	p := ui.NewStepPrinter(&buf)
	p.StartCheck(1, 12, "Git")
	p.StartInstall(1, 12, "Git")
	p.FinishInstall(1, 12, "Git", 2*time.Second)
	if buf.Len() == 0 {
		t.Error("expected output, got empty")
	}
}

func TestStepPrinter_FinishError(t *testing.T) {
	var buf bytes.Buffer
	p := ui.NewStepPrinter(&buf)
	p.StartCheck(1, 12, "Git")
	p.FinishError(1, 12, "Git", "brew install failed")
	if buf.Len() == 0 {
		t.Error("expected output, got empty")
	}
}
