package ui_test

import (
	"strings"
	"testing"

	"gearup/internal/ui"
)

func TestRenderPreview_AllInstalled(t *testing.T) {
	steps := []ui.StepStatus{
		{Index: 1, Total: 3, Name: "Homebrew", Installed: true},
		{Index: 2, Total: 3, Name: "Git", Installed: true},
		{Index: 3, Total: 3, Name: "jq", Installed: true},
	}
	out := ui.RenderPreview(steps)
	if !strings.Contains(out, "Homebrew") {
		t.Errorf("preview missing step name: %q", out)
	}
	if !strings.Contains(out, "already installed") || strings.Contains(out, "will install") {
		t.Errorf("status wrong: %q", out)
	}
	if !strings.Contains(out, "0 to install") {
		t.Errorf("summary wrong: %q", out)
	}
}

func TestRenderPreview_SomeWouldInstall(t *testing.T) {
	steps := []ui.StepStatus{
		{Index: 1, Total: 2, Name: "Homebrew", Installed: true},
		{Index: 2, Total: 2, Name: "nvm", Installed: false, RequiresElevation: true},
	}
	out := ui.RenderPreview(steps)
	if !strings.Contains(out, "will install") {
		t.Errorf("should say will install: %q", out)
	}
	if !strings.Contains(out, "elevation") {
		t.Errorf("should mention elevation: %q", out)
	}
	if !strings.Contains(out, "1 to install") {
		t.Errorf("summary wrong: %q", out)
	}
}

func TestRenderPreview_Empty(t *testing.T) {
	out := ui.RenderPreview(nil)
	if !strings.Contains(out, "0 to install") {
		t.Errorf("empty plan should show 0: %q", out)
	}
}
