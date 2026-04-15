# gearup Phase 3 — Elevation flow (first taste of Charm)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Support steps that require admin permissions, with a configurable pre-install banner + confirmation prompt. Introduce the first two Charm libraries (Lip Gloss + Huh) narrowly — full UI polish is Phase 4.

**Architecture:** New `config.Elevation` type on `Recipe`. New `internal/elevation` package with a `Prompter` interface (real: Huh; fake: for tests) and a `Timer` for duration warnings. Runner partitions steps into elevation-required vs regular; if a recipe has an `elevation:` block and at least one step declares `requires_elevation: true`, the runner acquires elevation via the banner + confirm, runs elevation-required steps back-to-back, then runs the rest. When no `elevation:` block is defined, runner ignores the flag and runs in declared order — steps that need sudo prompt the user natively via their shell command.

**Tech Stack:** Go, cobra, yaml.v3, plus new: `github.com/charmbracelet/lipgloss` (banner styling), `github.com/charmbracelet/huh` (confirmation prompt).

**Spec reference:** `docs/superpowers/specs/2026-04-15-gearup-design.md` §4.3 (elevation model), §5 Phase 3.

---

## Scope

**In Phase 3:**
- `Elevation{Message string; Duration time.Duration}` type, optional field on `Recipe`.
- `internal/elevation` package: `Prompter` interface, `HuhPrompter` real impl, `FakePrompter` test double, `Timer` for countdown/expiry, `Acquire(ctx, cfg, prompter, out) (*Timer, error)`.
- Lip Gloss styled banner rendering the configured message.
- Huh Confirm prompt ("Proceed?" → Yes/Abort).
- Runner partitions steps; acquires elevation once; runs elevation-required steps first while checking Timer for expiry warnings.
- JVM ingredient gains a real elevation-requiring step (symlink openjdk@21 into `/Library/Java/JavaVirtualMachines/`) to exercise the flow end-to-end.
- Backend recipe gains a generic `elevation:` block with a `duration: 180s` hint.
- Version bump to `0.0.4-phase3`; README updated.

**Out of Phase 3 (deferred to Phase 4):**
- Recipe picker (Huh multi-select).
- Plan preview (Lip Gloss table).
- Bubbles spinner during execution.
- `--dry-run` / `gearup plan`.
- Silencing check-command stdout leak.

---

## File structure (changes)

```
gearup/
├── internal/
│   ├── config/
│   │   ├── config.go                   # MODIFY: add Elevation type + field on Recipe
│   │   └── config_test.go              # MODIFY: add TestElevationUnmarshal
│   ├── elevation/
│   │   ├── elevation.go                # NEW: Prompter, HuhPrompter, Acquire
│   │   ├── timer.go                    # NEW: Timer
│   │   ├── fake.go                     # NEW: FakePrompter test double
│   │   └── elevation_test.go           # NEW: Acquire + Timer tests
│   └── runner/
│       ├── runner.go                   # MODIFY: partition + elevation integration
│       └── runner_test.go              # MODIFY: add elevation-flow tests
├── cmd/gearup/
│   └── main.go                         # MODIFY: wire HuhPrompter; bump version
├── examples/
│   ├── ingredients/
│   │   └── jvm.yaml                    # MODIFY: add JVM symlink step
│   └── recipes/
│       └── backend.yaml                # MODIFY: add elevation block
└── go.mod / go.sum                     # MODIFY: add lipgloss + huh deps
```

---

## Task 1: Add `Elevation` type to config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Append to `/Users/dlo/Dev/gearup/internal/config/config_test.go`:

```go
func TestElevationUnmarshal(t *testing.T) {
	const src = `
version: 1
name: "Backend"
elevation:
  message: "Please elevate admin permissions now, then press Enter."
  duration: 180s
`
	var r config.Recipe
	if err := yaml.Unmarshal([]byte(src), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Elevation == nil {
		t.Fatal("Elevation is nil, want non-nil")
	}
	if r.Elevation.Message == "" {
		t.Error("Elevation.Message empty")
	}
	if r.Elevation.Duration != 180*time.Second {
		t.Errorf("Elevation.Duration = %v, want 180s", r.Elevation.Duration)
	}
}
```

You'll also need to add `"time"` to the imports of `config_test.go` if not present.

- [ ] **Step 2: Run; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/config/...
```

- [ ] **Step 3: Add the type to `config.go`**

Edit `/Users/dlo/Dev/gearup/internal/config/config.go`. Add `"time"` import if not present. Add these declarations (place after the `Ingredient` type, before `ResolvedPlan`):

```go
// Elevation describes a recipe-level request for admin permissions.
// When a recipe has an Elevation block and at least one step declares
// RequiresElevation: true, the runner shows the Message in a banner,
// asks the user to confirm (assumed to have acquired elevation through
// whatever mechanism their org provides), then runs the elevation-required
// steps back-to-back. If Duration is non-zero, the runner warns if a
// subsequent elevation step is about to begin when less than 30s remain.
type Elevation struct {
	Message  string        `yaml:"message"`
	Duration time.Duration `yaml:"duration,omitempty"`
}
```

Then add the optional field to `Recipe` (place after `Platform`):

```go
	Elevation         *Elevation         `yaml:"elevation,omitempty"`
```

The pointer is deliberate: `nil` means "no elevation block declared" vs an empty struct which is ambiguous.

- [ ] **Step 4: Run; expect PASS**

```bash
go test ./internal/config/... -v
```

Expected: all config tests including the new `TestElevationUnmarshal` pass.

- [ ] **Step 5: Run full suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add Elevation type on Recipe"
```

---

## Task 2: Add Charm dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Fetch the deps**

```bash
cd /Users/dlo/Dev/gearup
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/huh@latest
go mod tidy
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: build succeeds. Nothing imports these yet, but they're in go.mod as direct requires.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add charmbracelet/lipgloss and charmbracelet/huh"
```

---

## Task 3: Timer implementation

**Files:**
- Create: `internal/elevation/timer.go`
- Create: `internal/elevation/timer_test.go`

- [ ] **Step 1: Failing test**

Create `/Users/dlo/Dev/gearup/internal/elevation/timer_test.go`:

```go
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
```

- [ ] **Step 2: Run; expect compile failure**

```bash
go test ./internal/elevation/...
```

- [ ] **Step 3: Implement Timer**

Create `/Users/dlo/Dev/gearup/internal/elevation/timer.go`:

```go
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
```

- [ ] **Step 4: Run; expect PASS**

```bash
go test ./internal/elevation/... -v
```

Expected: 4 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/elevation/
git commit -m "feat(elevation): add Timer for elevation-window tracking"
```

---

## Task 4: Prompter interface + FakePrompter + Acquire

**Files:**
- Create: `internal/elevation/elevation.go`
- Create: `internal/elevation/fake.go`
- Modify: `internal/elevation/elevation_test.go`

- [ ] **Step 1: Failing test**

Create `/Users/dlo/Dev/gearup/internal/elevation/elevation_test.go`:

```go
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
```

- [ ] **Step 2: Run; expect compile failure**

```bash
go test ./internal/elevation/...
```

- [ ] **Step 3: Implement FakePrompter**

Create `/Users/dlo/Dev/gearup/internal/elevation/fake.go`:

```go
package elevation

import "sync"

// FakePrompter is a test double for Prompter.
type FakePrompter struct {
	Result bool
	Err    error

	mu    sync.Mutex
	calls int
}

// NewFakePrompter returns an empty FakePrompter that returns (false, nil)
// by default. Set Result and/or Err to program its response.
func NewFakePrompter() *FakePrompter {
	return &FakePrompter{}
}

// Confirm records the call and returns the programmed response.
func (f *FakePrompter) Confirm(_ string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.Result, f.Err
}

// Calls returns how many times Confirm has been invoked.
func (f *FakePrompter) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}
```

- [ ] **Step 4: Implement Prompter, HuhPrompter, Acquire**

Create `/Users/dlo/Dev/gearup/internal/elevation/elevation.go`:

```go
package elevation

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"gearup/internal/config"
)

// Prompter is the interactive surface Acquire uses to ask the user to
// confirm they have acquired admin permissions.
type Prompter interface {
	// Confirm shows title and returns (true, nil) if the user affirms,
	// (false, nil) if they abort, (_, err) on any other failure.
	Confirm(title string) (bool, error)
}

// HuhPrompter is the production Prompter backed by charmbracelet/huh.
type HuhPrompter struct{}

// Confirm renders a Huh confirm prompt.
func (HuhPrompter) Confirm(title string) (bool, error) {
	var ok bool
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Continue").
		Negative("Abort").
		Value(&ok).
		Run()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// bannerStyle is the Lip Gloss style for elevation banners.
var bannerStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("214")). // amber-yellow
	Padding(1, 2).
	Bold(true)

// Acquire prints a styled banner containing cfg.Message, prompts the user
// for confirmation via the supplied Prompter, and returns a Timer started
// at the confirmation moment. Acquire returns an error if the user aborts
// or the Prompter fails.
func Acquire(_ context.Context, cfg *config.Elevation, p Prompter, out io.Writer) (*Timer, error) {
	if cfg == nil || cfg.Message == "" {
		return nil, fmt.Errorf("elevation: message is required")
	}

	fmt.Fprintln(out, bannerStyle.Render(cfg.Message))

	ok, err := p.Confirm("Proceed with elevation-required steps?")
	if err != nil {
		return nil, fmt.Errorf("elevation confirm: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("elevation aborted by user")
	}

	return NewTimer(cfg.Duration), nil
}
```

- [ ] **Step 5: Run; expect PASS**

```bash
go test ./internal/elevation/... -v
```

Expected: all 4 Acquire tests + 4 Timer tests pass (8 total).

- [ ] **Step 6: Run full suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 7: Commit**

```bash
git add internal/elevation/
git commit -m "feat(elevation): add Prompter interface, HuhPrompter, Acquire"
```

---

## Task 5: Runner integration (partition + elevation flow)

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/runner_test.go`

- [ ] **Step 1: Write failing tests**

Append these tests to `/Users/dlo/Dev/gearup/internal/runner/runner_test.go`:

```go
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
	// Order-checker installer: records the order its Install is called in.
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
	// Elevation steps b,d first, then regular a,c.
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
	// If a step has RequiresElevation:true but the Recipe has no Elevation
	// block, the runner does NOT partition and does NOT acquire elevation.
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
```

Add a test helper at the top of the test file (near `recordingInstaller`):

```go
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
```

Add imports to `runner_test.go` if not already present: `"strings"`, `"time"`, `"gearup/internal/elevation"`.

- [ ] **Step 2: Run; expect compile failure**

```bash
go test ./internal/runner/...
```

The existing tests should fail to compile because `runner.Runner` doesn't yet have a `Prompter` field.

- [ ] **Step 3: Update runner.go**

Replace the entire body of `/Users/dlo/Dev/gearup/internal/runner/runner.go` with:

```go
// Package runner executes a ResolvedPlan by dispatching each step to its
// registered installer. It stops on the first error (fail-fast). When the
// plan's Recipe declares an Elevation block and at least one step requires
// elevation, the runner batches elevation-required steps after acquiring
// confirmation from the user.
package runner

import (
	"context"
	"fmt"
	"time"

	"gearup/internal/config"
	"gearup/internal/elevation"
	"gearup/internal/installer"
)

// Writer is the minimal output surface the Runner needs.
type Writer interface {
	Printf(format string, args ...any)
	Write(p []byte) (int, error)
}

// Runner orchestrates plan execution.
type Runner struct {
	Registry installer.Registry
	Out      Writer
	Prompter elevation.Prompter // required if any step may require elevation
}

// expiryWarnThreshold is how soon before the elevation window ends we
// print a "running low" warning.
const expiryWarnThreshold = 30 * time.Second

// Run walks plan.Steps. If the Recipe has an Elevation block AND at least
// one step has RequiresElevation:true, elevation is acquired once up front,
// then elevation-required steps run back-to-back, then the remaining steps.
// Otherwise all steps run in declared order.
func (r *Runner) Run(ctx context.Context, plan *config.ResolvedPlan) error {
	elevSteps, regSteps := partition(plan)

	// Decide whether to open an elevation window.
	openWindow := len(elevSteps) > 0 &&
		plan.Recipe != nil &&
		plan.Recipe.Elevation != nil

	if openWindow {
		timer, err := elevation.Acquire(ctx, plan.Recipe.Elevation, r.Prompter, r)
		if err != nil {
			return err
		}
		for _, ix := range elevSteps {
			if timer.IsNearExpiry(expiryWarnThreshold) {
				r.Out.Printf("⚠  less than %s remain in elevation window\n", expiryWarnThreshold)
			}
			if err := r.runStep(ctx, ix.idx, ix.step, len(plan.Steps)); err != nil {
				return err
			}
		}
		for _, ix := range regSteps {
			if err := r.runStep(ctx, ix.idx, ix.step, len(plan.Steps)); err != nil {
				return err
			}
		}
		return nil
	}

	// No elevation window — run in declared order.
	for i, step := range plan.Steps {
		if err := r.runStep(ctx, i, step, len(plan.Steps)); err != nil {
			return err
		}
	}
	return nil
}

// Write satisfies io.Writer so elevation.Acquire can print its banner.
func (r *Runner) Write(p []byte) (int, error) {
	if w, ok := r.Out.(interface {
		Write(p []byte) (int, error)
	}); ok {
		return w.Write(p)
	}
	r.Out.Printf("%s", p)
	return len(p), nil
}

type indexedStep struct {
	idx  int
	step config.Step
}

// partition splits plan.Steps into elevation-required and regular groups,
// preserving declared order within each group. Indexes are 0-based and
// preserved for consistent [i/total] display.
func partition(plan *config.ResolvedPlan) (elev, reg []indexedStep) {
	for i, s := range plan.Steps {
		if s.RequiresElevation {
			elev = append(elev, indexedStep{idx: i, step: s})
		} else {
			reg = append(reg, indexedStep{idx: i, step: s})
		}
	}
	return
}

// runStep executes one step against its registered installer.
func (r *Runner) runStep(ctx context.Context, i int, step config.Step, total int) error {
	idx := i + 1
	inst, err := r.Registry.Get(step.Type)
	if err != nil {
		return fmt.Errorf("step %d (%s): %w", idx, step.Name, err)
	}
	r.Out.Printf("[%d/%d] %s: checking...\n", idx, total, step.Name)
	installed, err := inst.Check(ctx, step)
	if err != nil {
		return fmt.Errorf("step %d (%s) check failed: %w", idx, step.Name, err)
	}
	if installed {
		r.Out.Printf("[%d/%d] %s: already installed — skip\n", idx, total, step.Name)
		return nil
	}
	r.Out.Printf("[%d/%d] %s: installing...\n", idx, total, step.Name)
	if err := inst.Install(ctx, step); err != nil {
		return fmt.Errorf("step %d (%s) install failed: %w", idx, step.Name, err)
	}
	r.Out.Printf("[%d/%d] %s: installed\n", idx, total, step.Name)
	return nil
}
```

Note the `Writer` interface now includes `Write(p []byte) (int, error)` so the Runner itself can satisfy `io.Writer` for `elevation.Acquire`. The existing `bufWriter` test helper has a `bytes.Buffer` which already has `Write`; extend `bufWriter` with a passthrough:

Add to `runner_test.go` (modify the existing `bufWriter`):

```go
func (w *bufWriter) Write(p []byte) (int, error) { return w.b.Write(p) }
```

Also: the main.go `stdPrinter` type needs a `Write` method. Task 6 will update it.

- [ ] **Step 4: Run; expect PASS**

```bash
go test ./internal/runner/... -v
```

Expected: all prior tests + 4 new elevation-flow tests pass.

- [ ] **Step 5: Run full suite**

```bash
go test ./...
```

Expected: all packages pass except possibly `cmd/gearup` which may have a compile error if `stdPrinter` hasn't been updated. That's addressed in Task 6.

- [ ] **Step 6: Commit**

```bash
git add internal/runner/
git commit -m "feat(runner): partition elevation steps; acquire window via elevation.Acquire"
```

---

## Task 6: Wire HuhPrompter into the CLI + update stdPrinter

**Files:**
- Modify: `cmd/gearup/main.go`

- [ ] **Step 1: Update imports**

Open `/Users/dlo/Dev/gearup/cmd/gearup/main.go`. Add to the import block:

```go
	"gearup/internal/elevation"
```

- [ ] **Step 2: Extend `stdPrinter` to satisfy the updated Writer interface**

The existing `stdPrinter` has `Printf` only. Add a `Write` method so it satisfies `io.Writer` via the Runner's fallback path. Replace:

```go
// stdPrinter adapts os.Stdout to runner.Writer.
type stdPrinter struct{}

func (stdPrinter) Printf(format string, args ...any) { fmt.Printf(format, args...) }
```

with:

```go
// stdPrinter adapts os.Stdout to runner.Writer.
type stdPrinter struct{}

func (stdPrinter) Printf(format string, args ...any) { fmt.Printf(format, args...) }
func (stdPrinter) Write(p []byte) (int, error)       { return os.Stdout.Write(p) }
```

- [ ] **Step 3: Wire HuhPrompter into the Runner construction**

Find the Runner construction inside `runCmd()`:

```go
			r := &runner.Runner{Registry: reg, Out: stdPrinter{}}
```

Replace with:

```go
			r := &runner.Runner{
				Registry: reg,
				Out:      stdPrinter{},
				Prompter: elevation.HuhPrompter{},
			}
```

- [ ] **Step 4: Bump version**

Find:

```go
const version = "0.0.3-phase2"
```

Replace with:

```go
const version = "0.0.4-phase3"
```

- [ ] **Step 5: Build + version check**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
./gearup version
```

Expected: `gearup 0.0.4-phase3`.

- [ ] **Step 6: Run all tests**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 7: Smoke-run the existing backend recipe (no elevation block yet, should behave exactly as Phase 2)**

```bash
./gearup run --recipe ./examples/recipes/backend.yaml
```

Expected: every step reports `already installed — skip`. No elevation banner (no elevation block in the recipe yet; Task 7 adds it).

- [ ] **Step 8: Commit**

```bash
git add cmd/gearup/main.go
git commit -m "feat(cli): wire HuhPrompter; bump to 0.0.4-phase3"
```

---

## Task 7: Add a real elevation-requiring step to the JVM ingredient

**Files:**
- Modify: `examples/ingredients/jvm.yaml`

- [ ] **Step 1: Add the symlink step**

Replace the contents of `/Users/dlo/Dev/gearup/examples/ingredients/jvm.yaml` with:

```yaml
version: 1
name: jvm
description: "JVM development toolchain (OpenJDK 21) + system Java discovery linkage"

steps:
  - name: OpenJDK 21
    type: brew
    formula: openjdk@21

  # The openjdk@21 formula is keg-only — it's not symlinked into
  # /opt/homebrew, and the system's /usr/libexec/java_home doesn't find
  # it. Link it into /Library/Java/JavaVirtualMachines/ so java tooling
  # discovers it. This write requires admin permissions.
  - name: Link OpenJDK 21 for system Java discovery
    type: shell
    requires_elevation: true
    check: test -L /Library/Java/JavaVirtualMachines/openjdk-21.jdk
    install: |
      sudo ln -sfn /opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk \
        /Library/Java/JavaVirtualMachines/openjdk-21.jdk
```

- [ ] **Step 2: Smoke-test that the JVM ingredient still resolves**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/recipe/... -v
```

Expected: `TestResolve_BackendFixture` will FAIL because it now resolves to 12 steps (was 11). Fix the assertion in Task 8.

Wait — that's a regression the test will catch. Let's fix the assertion here and commit together. Continue with Step 3.

- [ ] **Step 3: Update the BackendFixture test step count**

In `/Users/dlo/Dev/gearup/internal/recipe/recipe_test.go`, find `TestResolve_BackendFixture`. Update any step-count assertion from `11` to `12`. Also update any assertions that reference step indexes after the new JVM symlink step (step index 3 is now the new symlink; kubectl is still found by name loop, so that assertion is safe).

Specifically:
- Change `if got, want := len(plan.Steps), 11; got != want { ... }` → `12`.
- Check `plan.Steps[10].Name == "nvm"` — this is no longer the last step. Update to `plan.Steps[11].Name == "nvm"` and `plan.Steps[11].Type == "curl-pipe-sh"`.
- `plan.Steps[2].Formula == "openjdk@21"` — this is still the OpenJDK step (index 2 unchanged; the symlink becomes index 3).

Add an explicit assertion that the symlink step exists:

```go
	// The JVM ingredient includes a symlink step that requires elevation.
	var jvmSymlink *config.Step
	for i := range plan.Steps {
		if plan.Steps[i].Name == "Link OpenJDK 21 for system Java discovery" {
			jvmSymlink = &plan.Steps[i]
			break
		}
	}
	if jvmSymlink == nil {
		t.Fatal("did not find JVM symlink step")
	}
	if !jvmSymlink.RequiresElevation {
		t.Error("jvm symlink step should have RequiresElevation:true")
	}
```

- [ ] **Step 4: Run and verify**

```bash
go test ./internal/recipe/... -v
```

Expected: `TestResolve_BackendFixture` passes with 12-step assertion. `TestResolve_FrontendFixture` is unchanged (still 4 steps).

- [ ] **Step 5: Full suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add examples/ingredients/jvm.yaml internal/recipe/recipe_test.go
git commit -m "feat(ingredients/jvm): add elevation-required JVM system symlink step"
```

---

## Task 8: Add `elevation:` block to the backend recipe

**Files:**
- Modify: `examples/recipes/backend.yaml`

- [ ] **Step 1: Add the block**

Replace contents of `/Users/dlo/Dev/gearup/examples/recipes/backend.yaml` with:

```yaml
version: 1
name: "Backend"
description: "Full macOS developer toolchain for backend/infra work"

platform:
  os: [darwin]

# If any step in this recipe requires elevation, gearup will show the
# message below and wait for your confirmation before running those steps.
# Ignore this block if you don't use an elevation workflow — gearup will
# fall back to native sudo prompts on a per-step basis.
elevation:
  message: "Some steps need admin permissions. Elevate your session now (via your usual mechanism), then press Continue."
  duration: 180s

ingredient_sources:
  - path: ../ingredients

ingredients:
  - base
  - jvm
  - containers
  - aws-k8s
  - node
```

- [ ] **Step 2: Update the `TestResolve_BackendFixture` to assert the elevation block is loaded**

Append inside the test:

```go
	// Phase 3: the backend recipe declares an elevation block.
	if p.Elevation == nil {
		t.Fatal("backend recipe has no Elevation block")
	}
	if p.Elevation.Duration == 0 {
		t.Error("backend recipe Elevation.Duration is zero")
	}
```

(Note: `p` is the variable name used for the loaded recipe in the test — if the variable is named differently, adjust accordingly.)

- [ ] **Step 3: Run**

```bash
go test ./internal/recipe/... -v
go test ./...
```

Expected: all packages pass.

- [ ] **Step 4: Commit**

```bash
git add examples/recipes/backend.yaml internal/recipe/recipe_test.go
git commit -m "feat(recipes/backend): add elevation block with 180s duration"
```

---

## Task 9: README update

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update status line**

Find:

```
Status: Phase 2 — in development. Supports `brew`, `curl-pipe-sh`, and `shell` step types on macOS.
```

Replace with:

```
Status: Phase 3 — in development. Supports three step types, per-step `requires_elevation` with a configurable banner + confirmation, on macOS.
```

- [ ] **Step 2: Add an Elevation section under Usage**

After the existing `## Usage` section's last paragraph, add:

```markdown

### Elevation

Steps that need admin permissions (e.g., writing to `/Library/...`) declare
`requires_elevation: true`. A recipe can include a top-level `elevation:`
block whose `message` is shown in a styled banner before such steps run —
e.g., "Please elevate admin permissions now, then press Continue." gearup
doesn't invoke elevation itself; it pauses and waits. If no `elevation:`
block is set, steps that need sudo prompt for a password natively as they
run.

See `examples/recipes/backend.yaml` for a full example, and
`examples/ingredients/jvm.yaml` for a step that requires elevation.
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document Phase 3 elevation flow"
```

---

## Task 10: Manual verification (user)

The user runs the backend recipe end-to-end on a Mac that already has everything else installed. Only the JVM symlink step should actually exercise the install path — and it needs sudo.

- [ ] **Step 1: Build**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
./gearup version
```

Expected: `gearup 0.0.4-phase3`.

- [ ] **Step 2: Confirm the JVM symlink isn't already in place**

```bash
ls -la /Library/Java/JavaVirtualMachines/openjdk-21.jdk 2>&1
```

If "No such file or directory" — good, the install path will fire. If a symlink is already there, remove it first with `sudo rm /Library/Java/JavaVirtualMachines/openjdk-21.jdk` so the verification exercises the install.

- [ ] **Step 3: Run the backend recipe**

```bash
./gearup run --recipe ./examples/recipes/backend.yaml
```

Expected behavior:
- 12 total steps this time.
- Elevation banner appears up front (yellow border, bold "Please elevate admin permissions..." message).
- Huh prompt: "Proceed with elevation-required steps? (Continue / Abort)". Hit Enter on "Continue".
- Step "Link OpenJDK 21 for system Java discovery" runs FIRST (despite being declared later in the jvm ingredient) because it's elevation-required.
- The `sudo ln -sfn ...` fires — macOS prompts for your password (native sudo).
- Then all other steps skip (already installed).
- `Done.`, exit 0.

- [ ] **Step 4: Verify the symlink exists**

```bash
ls -la /Library/Java/JavaVirtualMachines/openjdk-21.jdk
/usr/libexec/java_home -v 21
```

Expected: symlink points at `/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk`; `java_home` resolves to the JDK 21 path.

- [ ] **Step 5: Idempotency re-run**

```bash
./gearup run --recipe ./examples/recipes/backend.yaml; echo exit=$?
```

Expected: banner still shows (there's still an elevation-required step in the recipe), you hit Continue, then the JVM symlink reports `already installed — skip`. No sudo prompt. `exit=0`.

(Minor UX wart for Phase 4: banner shows even when there's nothing to actually do. Phase 4 with `--dry-run` will make this nicer by skipping the banner when all elevation steps already satisfy their check. Not blocking for Phase 3.)

- [ ] **Step 6: Frontend recipe sanity check**

The frontend recipe has NO elevation block and NO elevation-required steps. Run it and confirm no banner appears:

```bash
./gearup run --recipe ./examples/recipes/frontend.yaml; echo exit=$?
```

Expected: no banner, all steps skip, exit 0.

- [ ] **Step 7: Abort path sanity check**

Run the backend recipe again but hit "Abort" in the Huh confirmation:

```bash
./gearup run --recipe ./examples/recipes/backend.yaml
```

At the Huh prompt, arrow to "Abort" and press Enter. Expected:
- The tool exits with a non-zero code.
- No steps run.

---

## Phase 3 completion criteria

- [ ] `go test ./...` passes (8 packages including new `internal/elevation`).
- [ ] `gearup version` prints `gearup 0.0.4-phase3`.
- [ ] On the user's Mac, the backend recipe shows the elevation banner + Huh confirm, then the symlink step runs (sudo-prompted) and completes.
- [ ] Re-run is idempotent: symlink step reports `already installed — skip`.
- [ ] Frontend recipe runs with no banner.
- [ ] Aborting at the Huh prompt halts execution cleanly with a non-zero exit.
- [ ] No proprietary content anywhere in the repo.

---

## Known Phase 3 limitations (addressed later)

- Banner always shows when elevation block + any requires_elevation step exists, even if the Check says the step is already satisfied. Phase 4 (`--dry-run` + pre-resolution of checks) naturally fixes this: if no elevation-required step would actually run, skip the banner.
- Huh runs full-terminal mode (clears screen briefly). Fine for now; future polish may switch to inline prompts.
- No TTY check before invoking Huh. On non-TTY (CI, pipe), Huh errors out — we inherit that behavior. A proper TTY guard is a Phase 4 concern.

---

## Self-review

**Spec coverage:**
- ✅ `requires_elevation` flag respected by runner — Task 5
- ✅ `Elevation{Message, Duration}` loaded from recipe — Task 1
- ✅ Banner rendering (Lip Gloss) — Task 4
- ✅ Confirmation (Huh) — Tasks 4, 6
- ✅ Time-budget warnings — Task 3 + runner integration in Task 5
- ✅ Fallback to native sudo when no elevation block — Task 5 partition logic
- ✅ Real elevation-requiring example — Task 7
- ✅ Version bump + docs — Tasks 6, 9

**Placeholder scan:** No "TBD" / "TODO" / "similar to" in this plan.

**Type consistency:** `config.Elevation`, `elevation.Timer`, `elevation.Prompter`, `elevation.HuhPrompter`, `elevation.FakePrompter`, `elevation.Acquire`, `runner.Runner` with new `Prompter` field — names used consistently in every task that references them.

**Trade-offs intentionally deferred:**
- Interactive Huh means non-TTY invocation fails. This is acceptable for Phase 3; TTY guard lands in Phase 4.
- Banner displayed even when check-pass makes elevation moot. Noted in limitations; fixed by `--dry-run` infrastructure in Phase 4.
- Check-command stdout leak unchanged from Phase 1/2. Phase 4.
