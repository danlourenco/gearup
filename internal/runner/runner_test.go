package runner_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gearup/internal/config"
	"gearup/internal/elevation"
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
func (w *bufWriter) Write(p []byte) (int, error)       { return w.b.Write(p) }

// installerFunc adapts two function literals to the Installer interface.
// Used by ordering tests to capture call sequences.
type installerFunc struct {
	check   func(context.Context, config.Step) (bool, error)
	install func(context.Context, config.Step) error
}

func (f *installerFunc) Check(ctx context.Context, s config.Step) (bool, error) {
	return f.check(ctx, s)
}
func (f *installerFunc) Install(ctx context.Context, s config.Step) error {
	return f.install(ctx, s)
}

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

func TestRunner_AcquiresElevationWhenNeeded(t *testing.T) {
	inst := &recordingInstaller{checkInstalled: false}
	reg := installer.Registry{"brew": inst, "shell": inst}
	w := &bufWriter{}
	prompter := elevation.NewFakePrompter()
	prompter.Result = true

	r := &runner.Runner{
		Registry: reg,
		Out:      w,
		Prompter: prompter,
	}

	plan := &config.ResolvedPlan{
		Recipe: &config.Recipe{
			Name: "Backend",
			Elevation: &config.Elevation{
				Message:  "Please elevate",
				Duration: 180 * time.Second,
			},
		},
		Steps: []config.Step{
			{Name: "regular", Type: "brew", Formula: "jq"},
			{Name: "needs-elev", Type: "shell", Check: "true", Install: "true", RequiresElevation: true},
		},
	}

	if err := r.Run(context.Background(), plan); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if prompter.Calls() != 1 {
		t.Errorf("prompter called %d times, want 1", prompter.Calls())
	}
	if !strings.Contains(w.b.String(), "Please elevate") {
		t.Errorf("banner message not in output: %q", w.b.String())
	}
}

func TestRunner_ElevationStepsRunBeforeRegular(t *testing.T) {
	type call struct{ name string }
	var order []call
	ordered := &installerFunc{
		check: func(_ context.Context, s config.Step) (bool, error) { return false, nil },
		install: func(_ context.Context, s config.Step) error {
			order = append(order, call{name: s.Name})
			return nil
		},
	}
	reg := installer.Registry{"brew": ordered, "shell": ordered}
	prompter := elevation.NewFakePrompter()
	prompter.Result = true

	r := &runner.Runner{Registry: reg, Out: &bufWriter{}, Prompter: prompter}
	plan := &config.ResolvedPlan{
		Recipe: &config.Recipe{
			Elevation: &config.Elevation{Message: "elevate"},
		},
		Steps: []config.Step{
			{Name: "a-reg", Type: "brew", Formula: "a"},
			{Name: "b-elev", Type: "shell", Check: "false", Install: "true", RequiresElevation: true},
			{Name: "c-reg", Type: "brew", Formula: "c"},
			{Name: "d-elev", Type: "shell", Check: "false", Install: "true", RequiresElevation: true},
		},
	}
	if err := r.Run(context.Background(), plan); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"b-elev", "d-elev", "a-reg", "c-reg"}
	got := []string{}
	for _, c := range order {
		got = append(got, c.name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestRunner_NoElevationBlock_RunsInDeclaredOrder(t *testing.T) {
	type call struct{ name string }
	var order []call
	ordered := &installerFunc{
		check: func(_ context.Context, s config.Step) (bool, error) { return false, nil },
		install: func(_ context.Context, s config.Step) error {
			order = append(order, call{name: s.Name})
			return nil
		},
	}
	reg := installer.Registry{"brew": ordered, "shell": ordered}
	prompter := elevation.NewFakePrompter()

	r := &runner.Runner{Registry: reg, Out: &bufWriter{}, Prompter: prompter}
	plan := &config.ResolvedPlan{
		Recipe: &config.Recipe{}, // No Elevation block.
		Steps: []config.Step{
			{Name: "first", Type: "brew", Formula: "jq", RequiresElevation: true},
			{Name: "second", Type: "brew", Formula: "git"},
		},
	}
	if err := r.Run(context.Background(), plan); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if prompter.Calls() != 0 {
		t.Errorf("prompter called %d times, want 0 (no elevation block)", prompter.Calls())
	}
	got := []string{order[0].name, order[1].name}
	want := []string{"first", "second"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v (declared order)", got, want)
	}
}

func TestRunner_UserAbortsElevation(t *testing.T) {
	inst := &recordingInstaller{}
	reg := installer.Registry{"shell": inst}
	prompter := elevation.NewFakePrompter()
	prompter.Result = false // user aborts

	r := &runner.Runner{Registry: reg, Out: &bufWriter{}, Prompter: prompter}
	plan := &config.ResolvedPlan{
		Recipe: &config.Recipe{Elevation: &config.Elevation{Message: "elevate"}},
		Steps: []config.Step{
			{Name: "needs-elev", Type: "shell", Check: "false", Install: "true", RequiresElevation: true},
		},
	}
	err := r.Run(context.Background(), plan)
	if err == nil {
		t.Fatal("want error when user aborts elevation, got nil")
	}
	if inst.installCalls != 0 {
		t.Errorf("installCalls = %d, want 0 (no steps should run after abort)", inst.installCalls)
	}
}

func TestRunner_DryRun_NoStepsWouldRun_Exits0(t *testing.T) {
	inst := &recordingInstaller{checkInstalled: true}
	reg := installer.Registry{"brew": inst}
	w := &bufWriter{}

	r := &runner.Runner{Registry: reg, Out: w, DryRun: true}
	plan := makePlan(config.Step{Name: "jq", Type: "brew", Formula: "jq"})

	err := r.Run(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inst.installCalls != 0 {
		t.Errorf("installCalls = %d, want 0 (dry-run should never install)", inst.installCalls)
	}
	if !strings.Contains(w.b.String(), "already installed") {
		t.Errorf("output should say already installed: %q", w.b.String())
	}
}

func TestRunner_DryRun_StepsWouldRun_ReturnsErrDryRunPending(t *testing.T) {
	inst := &recordingInstaller{checkInstalled: false}
	reg := installer.Registry{"brew": inst}
	w := &bufWriter{}

	r := &runner.Runner{Registry: reg, Out: w, DryRun: true}
	plan := makePlan(config.Step{Name: "jq", Type: "brew", Formula: "jq"})

	err := r.Run(context.Background(), plan)
	if err == nil {
		t.Fatal("want ErrDryRunPending, got nil")
	}
	if !errors.Is(err, runner.ErrDryRunPending) {
		t.Errorf("err = %v, want ErrDryRunPending", err)
	}
	if inst.installCalls != 0 {
		t.Errorf("installCalls = %d, want 0 (dry-run should never install)", inst.installCalls)
	}
	if !strings.Contains(w.b.String(), "WOULD install") {
		t.Errorf("output should say WOULD install: %q", w.b.String())
	}
}

func TestRunner_DryRun_SkipsElevationAcquire(t *testing.T) {
	inst := &recordingInstaller{checkInstalled: false}
	reg := installer.Registry{"shell": inst}
	prompter := elevation.NewFakePrompter()
	w := &bufWriter{}

	r := &runner.Runner{Registry: reg, Out: w, Prompter: prompter, DryRun: true}
	plan := &config.ResolvedPlan{
		Recipe: &config.Recipe{
			Elevation: &config.Elevation{Message: "elevate"},
		},
		Steps: []config.Step{
			{Name: "needs-elev", Type: "shell", Check: "false", Install: "true", RequiresElevation: true},
		},
	}

	err := r.Run(context.Background(), plan)
	if !errors.Is(err, runner.ErrDryRunPending) {
		t.Fatalf("err = %v, want ErrDryRunPending", err)
	}
	if prompter.Calls() != 0 {
		t.Errorf("prompter called %d times in dry-run, want 0", prompter.Calls())
	}
}
