package runner_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gearup/internal/config"
	"gearup/internal/installer"
	"gearup/internal/runner"
)

// recordingInstaller tracks calls and returns programmed results.
type recordingInstaller struct {
	checkInstalled bool
	checkErr       error
	installErr     error
	checkCalls     int
	installCalls   int
}

func (r *recordingInstaller) Check(_ context.Context, _ config.Step) (bool, error) {
	r.checkCalls++
	return r.checkInstalled, r.checkErr
}
func (r *recordingInstaller) Install(_ context.Context, _ config.Step) error {
	r.installCalls++
	return r.installErr
}

// bufWriter captures Printf output for assertions.
type bufWriter struct{ b bytes.Buffer }

func (w *bufWriter) Printf(format string, args ...any) { fmt.Fprintf(&w.b, format, args...) }

func makePlan(steps ...config.Step) *config.ResolvedPlan {
	return &config.ResolvedPlan{Steps: steps}
}

func TestRunner_SkipsAlreadyInstalled(t *testing.T) {
	inst := &recordingInstaller{checkInstalled: true}
	reg := installer.Registry{"brew": inst}
	w := &bufWriter{}
	r := &runner.Runner{Registry: reg, Out: w}

	plan := makePlan(config.Step{Name: "jq", Type: "brew", Formula: "jq"})
	if err := r.Run(context.Background(), plan); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if inst.checkCalls != 1 {
		t.Errorf("checkCalls = %d, want 1", inst.checkCalls)
	}
	if inst.installCalls != 0 {
		t.Errorf("installCalls = %d, want 0 (should skip)", inst.installCalls)
	}
	if !strings.Contains(w.b.String(), "already installed") {
		t.Errorf("output = %q, want contains 'already installed'", w.b.String())
	}
}

func TestRunner_InstallsWhenNotInstalled(t *testing.T) {
	inst := &recordingInstaller{checkInstalled: false}
	reg := installer.Registry{"brew": inst}
	w := &bufWriter{}
	r := &runner.Runner{Registry: reg, Out: w}

	plan := makePlan(config.Step{Name: "jq", Type: "brew", Formula: "jq"})
	if err := r.Run(context.Background(), plan); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if inst.installCalls != 1 {
		t.Errorf("installCalls = %d, want 1", inst.installCalls)
	}
}

func TestRunner_StopsOnCheckError(t *testing.T) {
	inst := &recordingInstaller{checkErr: errors.New("boom")}
	reg := installer.Registry{"brew": inst}
	w := &bufWriter{}
	r := &runner.Runner{Registry: reg, Out: w}

	plan := makePlan(
		config.Step{Name: "jq", Type: "brew", Formula: "jq"},
		config.Step{Name: "git", Type: "brew", Formula: "git"},
	)
	err := r.Run(context.Background(), plan)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if inst.checkCalls != 1 {
		t.Errorf("checkCalls = %d, want 1 (should stop)", inst.checkCalls)
	}
}

func TestRunner_StopsOnInstallError(t *testing.T) {
	inst := &recordingInstaller{checkInstalled: false, installErr: errors.New("install failed")}
	reg := installer.Registry{"brew": inst}
	w := &bufWriter{}
	r := &runner.Runner{Registry: reg, Out: w}

	plan := makePlan(
		config.Step{Name: "jq", Type: "brew", Formula: "jq"},
		config.Step{Name: "git", Type: "brew", Formula: "git"},
	)
	err := r.Run(context.Background(), plan)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if inst.installCalls != 1 {
		t.Errorf("installCalls = %d, want 1 (should stop)", inst.installCalls)
	}
}

func TestRunner_UnknownStepType(t *testing.T) {
	reg := installer.Registry{}
	w := &bufWriter{}
	r := &runner.Runner{Registry: reg, Out: w}

	plan := makePlan(config.Step{Name: "x", Type: "nonsense"})
	err := r.Run(context.Background(), plan)
	if err == nil {
		t.Fatal("want error for unknown step type, got nil")
	}
}
