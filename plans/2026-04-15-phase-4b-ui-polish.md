# gearup Phase 4B — UI polish (picker, preview, spinner)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the raw text UX with Charm-native interactive elements: a Huh recipe picker when `--recipe` is omitted, a Lip Gloss plan preview before execution (and as `gearup plan` output), a Bubbles spinner during step execution, and smart elevation-banner suppression when no elevation step would actually fire.

**Architecture:** New `internal/ui` package owns the spinner, plan-preview renderer, and recipe-picker. The runner emits structured events that the UI layer consumes. The CLI wires the UI layer into the runner and recipe-resolution paths.

**Tech Stack:** Go, cobra, yaml.v3, `charmbracelet/lipgloss` (styling), `charmbracelet/huh` (picker + confirm), `charmbracelet/bubbles/spinner` (animated status).

**Spec reference:** `docs/superpowers/specs/2026-04-15-gearup-design.md` §4.1–4.2, §5 Phase 4.

---

## Scope

**In Phase 4B:**
- Huh recipe picker: scan `$XDG_CONFIG_HOME/gearup/recipes/` + `./examples/recipes/` for YAML files, parse name + description, present a Huh select. Auto-select if exactly one recipe is found.
- Lip Gloss plan preview: styled table showing step number, name, status (will install / already installed), and elevation annotation. Summary footer. Rendered before execution in `run` (auto-proceed, no second confirm) and as the primary output of `plan`.
- Bubbles spinner: single live-updating line per step during execution. Completed steps rendered as static `✓` or `✗` lines above. Current step shows animated spinner + elapsed time.
- Elevation banner suppression: pre-check elevation-required steps before showing the banner. If all already pass their `check`, skip the banner and Huh confirm entirely.
- `go get github.com/charmbracelet/bubbles` for the spinner.

**Out of Phase 4B:**
- Log rotation (deferred indefinitely per spec).
- Recipe-picker multi-select (only single-select for now).
- Progress bar across all steps (spinner per step is sufficient).

---

## File structure (changes)

```
gearup/
├── internal/
│   ├── ui/
│   │   ├── picker.go              # NEW: Huh recipe picker
│   │   ├── picker_test.go         # NEW
│   │   ├── preview.go             # NEW: Lip Gloss plan preview table
│   │   ├── preview_test.go        # NEW
│   │   ├── spinner.go             # NEW: Bubbles spinner for step execution
│   │   └── spinner_test.go        # NEW (basic unit tests; visual tested manually)
│   ├── runner/
│   │   ├── runner.go              # MODIFY: emit events instead of Printf; pre-check elevation
│   │   └── runner_test.go         # MODIFY: update for new event-driven output
│   └── elevation/
│       └── elevation.go           # MODIFY (minor): Acquire takes a skip-if-unnecessary flag
├── cmd/gearup/
│   └── main.go                    # MODIFY: wire picker, preview, spinner; version bump
└── go.mod / go.sum                # MODIFY: add bubbles dep
```

---

## Task 1: Add Bubbles dependency

- [ ] **Step 1: Fetch**

```bash
cd /Users/dlo/Dev/gearup
go get github.com/charmbracelet/bubbles@latest
go mod tidy
go build ./...
```

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add charmbracelet/bubbles"
```

---

## Task 2: Recipe picker (`internal/ui/picker.go`)

**Files:**
- Create: `internal/ui/picker.go`
- Create: `internal/ui/picker_test.go`

- [ ] **Step 1: Failing test**

Create `/Users/dlo/Dev/gearup/internal/ui/picker_test.go`:

```go
package ui_test

import (
	"os"
	"path/filepath"
	"testing"

	"gearup/internal/ui"
)

func writeRecipe(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestDiscoverRecipes_FindsYAML(t *testing.T) {
	dir := t.TempDir()
	writeRecipe(t, dir, "backend.yaml", `
version: 1
name: Backend
description: "Backend toolchain"
`)
	writeRecipe(t, dir, "frontend.yaml", `
version: 1
name: Frontend
description: "Frontend toolchain"
`)
	// Non-yaml file should be ignored.
	writeRecipe(t, dir, "notes.txt", "not a recipe")

	recipes, err := ui.DiscoverRecipes([]string{dir})
	if err != nil {
		t.Fatalf("DiscoverRecipes: %v", err)
	}
	if len(recipes) != 2 {
		t.Fatalf("got %d recipes, want 2", len(recipes))
	}
}

func TestDiscoverRecipes_EmptyDirs(t *testing.T) {
	dir := t.TempDir()
	recipes, err := ui.DiscoverRecipes([]string{dir})
	if err != nil {
		t.Fatalf("DiscoverRecipes: %v", err)
	}
	if len(recipes) != 0 {
		t.Errorf("got %d recipes, want 0", len(recipes))
	}
}

func TestDiscoverRecipes_SkipsMissingDirs(t *testing.T) {
	recipes, err := ui.DiscoverRecipes([]string{"/nonexistent/path/gearup"})
	if err != nil {
		t.Fatalf("DiscoverRecipes: %v", err)
	}
	if len(recipes) != 0 {
		t.Errorf("got %d recipes, want 0", len(recipes))
	}
}

func TestDiscoverRecipes_DeduplicatesByPath(t *testing.T) {
	dir := t.TempDir()
	writeRecipe(t, dir, "backend.yaml", `
version: 1
name: Backend
description: "Backend toolchain"
`)
	// Pass the same directory twice.
	recipes, err := ui.DiscoverRecipes([]string{dir, dir})
	if err != nil {
		t.Fatalf("DiscoverRecipes: %v", err)
	}
	if len(recipes) != 1 {
		t.Errorf("got %d recipes, want 1 (deduped)", len(recipes))
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

- [ ] **Step 3: Implementation**

Create `/Users/dlo/Dev/gearup/internal/ui/picker.go`:

```go
// Package ui provides interactive terminal elements backed by the Charm
// ecosystem: recipe picker (Huh), plan preview (Lip Gloss), and execution
// spinner (Bubbles).
package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"
)

// RecipeEntry is a discovered recipe file with its parsed metadata.
type RecipeEntry struct {
	Path        string
	Name        string
	Description string
}

// DiscoverRecipes scans dirs for *.yaml files, parses their name and
// description fields, and returns a deduplicated list. Missing directories
// are silently skipped.
func DiscoverRecipes(dirs []string) ([]RecipeEntry, error) {
	seen := map[string]bool{}
	var entries []RecipeEntry

	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			return nil, err
		}
		for _, path := range matches {
			abs, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			if seen[abs] {
				continue
			}
			seen[abs] = true
			entry, err := parseRecipeEntry(abs)
			if err != nil {
				continue // skip unparseable files
			}
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func parseRecipeEntry(path string) (RecipeEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RecipeEntry{}, err
	}
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return RecipeEntry{}, err
	}
	if meta.Name == "" {
		meta.Name = filepath.Base(path)
	}
	return RecipeEntry{Path: path, Name: meta.Name, Description: meta.Description}, nil
}

// PickRecipe shows an interactive Huh select for the given entries.
// If exactly one entry exists, it is returned without prompting.
// Returns the selected entry or an error if the user aborts.
func PickRecipe(entries []RecipeEntry) (RecipeEntry, error) {
	if len(entries) == 0 {
		return RecipeEntry{}, fmt.Errorf("no recipes found; pass --recipe <path> explicitly")
	}
	if len(entries) == 1 {
		return entries[0], nil
	}

	options := make([]huh.Option[int], len(entries))
	for i, e := range entries {
		label := e.Name
		if e.Description != "" {
			label = fmt.Sprintf("%s — %s", e.Name, e.Description)
		}
		options[i] = huh.NewOption(label, i)
	}

	var selected int
	err := huh.NewSelect[int]().
		Title("Which recipe?").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return RecipeEntry{}, fmt.Errorf("recipe picker: %w", err)
	}
	return entries[selected], nil
}
```

- [ ] **Step 4: Run tests; expect PASS**

```bash
go test ./internal/ui/... -v
```

Expected: 4 tests pass (DiscoverRecipes tests only; PickRecipe requires TTY and is tested manually).

- [ ] **Step 5: Commit**

```bash
git add internal/ui/
git commit -m "feat(ui): add recipe discovery and Huh picker"
```

---

## Task 3: Plan preview (`internal/ui/preview.go`)

**Files:**
- Create: `internal/ui/preview.go`
- Create: `internal/ui/preview_test.go`

- [ ] **Step 1: Failing test**

Create `/Users/dlo/Dev/gearup/internal/ui/preview_test.go`:

```go
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
```

- [ ] **Step 2: Run; expect compile failure**

- [ ] **Step 3: Implementation**

Create `/Users/dlo/Dev/gearup/internal/ui/preview.go`:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StepStatus holds the pre-check result for one step, used by the plan preview.
type StepStatus struct {
	Index             int
	Total             int
	Name              string
	Installed         bool
	RequiresElevation bool
}

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	arrow     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("→")
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	boldStyle = lipgloss.NewStyle().Bold(true)
	elevTag   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("[elevation]")
)

// RenderPreview returns a styled multi-line plan preview string.
func RenderPreview(steps []StepStatus) string {
	var b strings.Builder
	toInstall := 0
	needsElev := 0

	for _, s := range steps {
		prefix := fmt.Sprintf("[%d/%d]", s.Index, s.Total)
		if s.Installed {
			fmt.Fprintf(&b, "  %s %s %s  %s\n",
				dimStyle.Render(prefix),
				checkMark,
				s.Name,
				dimStyle.Render("already installed"),
			)
		} else {
			toInstall++
			elev := ""
			if s.RequiresElevation {
				needsElev++
				elev = "  " + elevTag
			}
			fmt.Fprintf(&b, "  %s %s %s  %s%s\n",
				prefix,
				arrow,
				boldStyle.Render(s.Name),
				"will install",
				elev,
			)
		}
	}

	total := len(steps)
	summary := fmt.Sprintf("\n  %d to install · %d already installed",
		toInstall, total-toInstall)
	if needsElev > 0 {
		summary += fmt.Sprintf(" · %d requires elevation", needsElev)
	}
	b.WriteString(summary + "\n")

	return b.String()
}
```

- [ ] **Step 4: Run; expect PASS**

```bash
go test ./internal/ui/... -v
```

Expected: all 4 DiscoverRecipes tests + 3 RenderPreview tests pass (7 total).

- [ ] **Step 5: Commit**

```bash
git add internal/ui/
git commit -m "feat(ui): add Lip Gloss plan preview renderer"
```

---

## Task 4: Step spinner (`internal/ui/spinner.go`)

**Files:**
- Create: `internal/ui/spinner.go`
- Create: `internal/ui/spinner_test.go`

- [ ] **Step 1: Failing test**

Create `/Users/dlo/Dev/gearup/internal/ui/spinner_test.go`:

```go
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
	out := buf.String()
	if len(out) == 0 {
		t.Error("expected output, got empty")
	}
}

func TestStepPrinter_CheckInstall(t *testing.T) {
	var buf bytes.Buffer
	p := ui.NewStepPrinter(&buf)
	p.StartCheck(1, 12, "Git")
	p.StartInstall(1, 12, "Git")
	p.FinishInstall(1, 12, "Git", 2*time.Second)
	out := buf.String()
	if len(out) == 0 {
		t.Error("expected output, got empty")
	}
}

func TestStepPrinter_FinishError(t *testing.T) {
	var buf bytes.Buffer
	p := ui.NewStepPrinter(&buf)
	p.StartCheck(1, 12, "Git")
	p.FinishError(1, 12, "Git", "brew install failed")
	out := buf.String()
	if len(out) == 0 {
		t.Error("expected output, got empty")
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

- [ ] **Step 3: Implementation**

Create `/Users/dlo/Dev/gearup/internal/ui/spinner.go`:

```go
package ui

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	successIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	failIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗")
	spinChars   = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
)

// StepPrinter renders step-execution status lines. It writes completed-step
// lines as permanent output, and can be extended to animate the current step
// with a spinner in a future refinement. For Phase 4B it uses a static
// single-line approach that's still visually cleaner than the Phase 4A
// multi-line output.
type StepPrinter struct {
	out io.Writer
}

// NewStepPrinter creates a StepPrinter that writes to out.
func NewStepPrinter(out io.Writer) *StepPrinter {
	return &StepPrinter{out: out}
}

// StartCheck prints the checking line for a step.
func (p *StepPrinter) StartCheck(idx, total int, name string) {
	fmt.Fprintf(p.out, "  %s [%d/%d] %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("○"),
		idx, total, name)
}

// FinishSkip overwrites the current step with a dimmed skip line.
func (p *StepPrinter) FinishSkip(idx, total int, name string) {
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		successIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("already installed"))
}

// StartInstall prints the installing line for a step.
func (p *StepPrinter) StartInstall(idx, total int, name string) {
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(string(spinChars[0])),
		idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("installing..."))
}

// FinishInstall overwrites the current step with a success line + elapsed time.
func (p *StepPrinter) FinishInstall(idx, total int, name string, elapsed time.Duration) {
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		successIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(
			fmt.Sprintf("installed (%s)", elapsed.Round(100*time.Millisecond))))
}

// FinishError overwrites the current step with a failure line.
func (p *StepPrinter) FinishError(idx, total int, name, errMsg string) {
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		failIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("FAILED: "+errMsg))
}
```

- [ ] **Step 4: Run; expect PASS**

```bash
go test ./internal/ui/... -v
```

Expected: all previous tests + 3 spinner tests pass (10 total).

- [ ] **Step 5: Commit**

```bash
git add internal/ui/
git commit -m "feat(ui): add step printer with check/skip/install/error rendering"
```

---

## Task 5: Runner refactor — emit structured events for the UI

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/runner_test.go`

The runner currently calls `r.Out.Printf(...)` directly. We need to give it a `StepPrinter` so it can call the semantic methods (`StartCheck`, `FinishSkip`, etc.) instead. Also: pre-check elevation-required steps before showing the banner.

- [ ] **Step 1: Add `UI` field and pre-check elevation**

Replace the entire contents of `/Users/dlo/Dev/gearup/internal/runner/runner.go` with:

```go
// Package runner executes a ResolvedPlan by dispatching each step to its
// registered installer. It stops on the first error (fail-fast).
package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gearup/internal/config"
	"gearup/internal/elevation"
	"gearup/internal/installer"
	"gearup/internal/ui"
)

// ErrDryRunPending is returned by Run when DryRun is true and at least one
// step reports it would be installed.
var ErrDryRunPending = errors.New("dry-run: one or more steps would install")

// Writer is the output surface for non-step messages (banner, summary).
type Writer interface {
	Printf(format string, args ...any)
	Write(p []byte) (int, error)
}

// Runner orchestrates plan execution.
type Runner struct {
	Registry installer.Registry
	Out      Writer
	Prompter elevation.Prompter
	Printer  *ui.StepPrinter // if nil, falls back to plain Printf on Out
	DryRun   bool
}

const expiryWarnThreshold = 30 * time.Second

// Run walks plan.Steps.
func (r *Runner) Run(ctx context.Context, plan *config.ResolvedPlan) error {
	if r.DryRun {
		return r.runDryRun(ctx, plan)
	}
	return r.runLive(ctx, plan)
}

func (r *Runner) runDryRun(ctx context.Context, plan *config.ResolvedPlan) error {
	total := len(plan.Steps)
	var statuses []ui.StepStatus
	willInstall := 0

	for i, step := range plan.Steps {
		idx := i + 1
		inst, err := r.Registry.Get(step.Type)
		if err != nil {
			return fmt.Errorf("step %d (%s): %w", idx, step.Name, err)
		}
		installed, err := inst.Check(ctx, step)
		if err != nil {
			return fmt.Errorf("step %d (%s) check failed: %w", idx, step.Name, err)
		}
		statuses = append(statuses, ui.StepStatus{
			Index:             idx,
			Total:             total,
			Name:              step.Name,
			Installed:         installed,
			RequiresElevation: step.RequiresElevation,
		})
		if !installed {
			willInstall++
		}
	}

	r.Out.Printf("%s", ui.RenderPreview(statuses))

	if willInstall > 0 {
		return ErrDryRunPending
	}
	return nil
}

func (r *Runner) runLive(ctx context.Context, plan *config.ResolvedPlan) error {
	elevSteps, regSteps := partition(plan)

	// Pre-check: do any elevation-required steps actually need to run?
	needsElevation := false
	if len(elevSteps) > 0 && plan.Recipe != nil && plan.Recipe.Elevation != nil {
		for _, ix := range elevSteps {
			inst, err := r.Registry.Get(ix.step.Type)
			if err != nil {
				return fmt.Errorf("step %d (%s): %w", ix.idx+1, ix.step.Name, err)
			}
			installed, err := inst.Check(ctx, ix.step)
			if err != nil {
				return fmt.Errorf("step %d (%s) check failed: %w", ix.idx+1, ix.step.Name, err)
			}
			if !installed {
				needsElevation = true
				break
			}
		}
	}

	if needsElevation {
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

	// No elevation needed — run in declared order (elevation steps already checked above,
	// but that's fine — Check is idempotent and cheap).
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

func (r *Runner) runStep(ctx context.Context, i int, step config.Step, total int) error {
	idx := i + 1
	inst, err := r.Registry.Get(step.Type)
	if err != nil {
		return fmt.Errorf("step %d (%s): %w", idx, step.Name, err)
	}

	if r.Printer != nil {
		r.Printer.StartCheck(idx, total, step.Name)
	} else {
		r.Out.Printf("[%d/%d] %s: checking...\n", idx, total, step.Name)
	}

	installed, err := inst.Check(ctx, step)
	if err != nil {
		if r.Printer != nil {
			r.Printer.FinishError(idx, total, step.Name, err.Error())
		}
		return fmt.Errorf("step %d (%s) check failed: %w", idx, step.Name, err)
	}
	if installed {
		if r.Printer != nil {
			r.Printer.FinishSkip(idx, total, step.Name)
		} else {
			r.Out.Printf("[%d/%d] %s: already installed — skip\n", idx, total, step.Name)
		}
		return nil
	}

	start := time.Now()
	if r.Printer != nil {
		r.Printer.StartInstall(idx, total, step.Name)
	} else {
		r.Out.Printf("[%d/%d] %s: installing...\n", idx, total, step.Name)
	}

	if err := inst.Install(ctx, step); err != nil {
		if r.Printer != nil {
			r.Printer.FinishError(idx, total, step.Name, err.Error())
		}
		return fmt.Errorf("step %d (%s) install failed: %w", idx, step.Name, err)
	}

	if r.Printer != nil {
		r.Printer.FinishInstall(idx, total, step.Name, time.Since(start))
	} else {
		r.Out.Printf("[%d/%d] %s: installed\n", idx, total, step.Name)
	}
	return nil
}
```

- [ ] **Step 2: Update tests**

The existing `runner_test.go` tests use `r.Out` (`bufWriter`) for assertions. Since `Printer` is nil in those tests, the runner falls back to `Printf` — existing tests should still pass without changes. Verify:

```bash
go test ./internal/runner/... -v
```

If they pass, no test changes needed.

Add one new test to verify the elevation-banner suppression:

Append to `internal/runner/runner_test.go`:

```go
func TestRunner_ElevationBannerSuppressedWhenAllElevStepsInstalled(t *testing.T) {
	inst := &recordingInstaller{checkInstalled: true}
	reg := installer.Registry{"shell": inst}
	prompter := elevation.NewFakePrompter()
	prompter.Result = true
	w := &bufWriter{}

	r := &runner.Runner{Registry: reg, Out: w, Prompter: prompter}
	plan := &config.ResolvedPlan{
		Recipe: &config.Recipe{
			Elevation: &config.Elevation{Message: "elevate now"},
		},
		Steps: []config.Step{
			{Name: "elev-step", Type: "shell", Check: "true", Install: "true", RequiresElevation: true},
		},
	}

	if err := r.Run(context.Background(), plan); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if prompter.Calls() != 0 {
		t.Errorf("prompter called %d times, want 0 (banner should be suppressed)", prompter.Calls())
	}
	if strings.Contains(w.b.String(), "elevate now") {
		t.Errorf("banner message should NOT appear: %q", w.b.String())
	}
}
```

Run again:

```bash
go test ./internal/runner/... -v
```

All tests pass.

- [ ] **Step 3: Full suite**

```bash
go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/runner/
git commit -m "feat(runner): use StepPrinter for UI; suppress banner when no elevation needed"
```

---

## Task 6: CLI wiring — picker, preview, spinner, version

**Files:**
- Modify: `cmd/gearup/main.go`

- [ ] **Step 1: Replace `cmd/gearup/main.go` with exactly:**

```go
// Command gearup is an open-source macOS developer-machine bootstrap CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"gearup/internal/elevation"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer"
	"gearup/internal/installer/brew"
	"gearup/internal/installer/curlpipe"
	installshell "gearup/internal/installer/shell"
	gearlog "gearup/internal/log"
	"gearup/internal/recipe"
	"gearup/internal/runner"
	"gearup/internal/ui"
)

const version = "0.0.6-phase4b"

func main() {
	root := &cobra.Command{
		Use:   "gearup",
		Short: "Open-source macOS developer-machine bootstrap CLI",
	}
	root.AddCommand(runCmd(), planCmd(), versionCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCmd() *cobra.Command {
	var recipePath string
	var dryRun, yes bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a provisioning recipe",
		RunE: func(c *cobra.Command, args []string) error {
			return execute(recipePath, dryRun, yes)
		},
	}
	cmd.Flags().StringVar(&recipePath, "recipe", "", "path to recipe YAML (omit to pick interactively)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "resolve checks without installing; exit 10 if anything would run")
	cmd.Flags().BoolVar(&yes, "yes", false, "auto-approve elevation confirmations (for scripted use)")
	return cmd
}

func planCmd() *cobra.Command {
	var recipePath string
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Alias for `run --dry-run`",
		RunE: func(c *cobra.Command, args []string) error {
			return execute(recipePath, true, true)
		},
	}
	cmd.Flags().StringVar(&recipePath, "recipe", "", "path to recipe YAML (omit to pick interactively)")
	return cmd
}

func execute(recipePath string, dryRun, yes bool) error {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "gearup currently supports macOS only")
		os.Exit(4)
	}

	// If no recipe specified, try to discover and pick one interactively.
	if recipePath == "" {
		picked, err := discoverAndPick()
		if err != nil {
			return err
		}
		recipePath = picked
	}

	// TTY guard: interactive runs (non-dry-run, non-yes) require a terminal.
	if !dryRun && !yes && !isTerminal(os.Stdin) {
		fmt.Fprintln(os.Stderr, "gearup requires an interactive terminal. Use --yes to bypass elevation prompts, or --dry-run to preview.")
		os.Exit(3)
	}

	absRecipe, err := filepath.Abs(recipePath)
	if err != nil {
		return err
	}
	rec, err := recipe.LoadRecipe(absRecipe)
	if err != nil {
		return err
	}
	plan, err := recipe.Resolve(rec, filepath.Dir(absRecipe))
	if err != nil {
		return err
	}

	// Open a per-run log file.
	lf, err := gearlog.Create(rec.Name)
	if err != nil {
		return err
	}
	defer lf.Close()

	shellRunner := &gearexec.ShellRunner{
		StreamOut: os.Stdout,
		StreamErr: os.Stderr,
		LogOut:    lf.Writer(),
	}

	reg := installer.Registry{
		"brew":         &brew.Installer{Runner: shellRunner},
		"curl-pipe-sh": &curlpipe.Installer{Runner: shellRunner},
		"shell":        &installshell.Installer{Runner: shellRunner},
	}

	var prompter elevation.Prompter = elevation.HuhPrompter{}
	if yes {
		prompter = elevation.AutoApprovePrompter{}
	}

	printer := ui.NewStepPrinter(os.Stdout)

	r := &runner.Runner{
		Registry: reg,
		Out:      stdPrinter{},
		Prompter: prompter,
		Printer:  printer,
		DryRun:   dryRun,
	}

	header := "RECIPE"
	if dryRun {
		header = "PLAN (dry-run)"
	}
	fmt.Printf("%s: %s  (%d steps)\n\n", header, rec.Name, len(plan.Steps))

	err = r.Run(context.Background(), plan)
	if errors.Is(err, runner.ErrDryRunPending) {
		fmt.Fprintln(os.Stderr, "\nnote: re-run without --dry-run to apply the pending installs")
		os.Exit(10)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nerror:", err)
		fmt.Fprintln(os.Stderr, "full log:", lf.Path())
		os.Exit(1)
	}
	if !dryRun {
		fmt.Println("\nDone.")
	}
	return nil
}

// discoverAndPick scans well-known directories for recipe files and
// prompts the user to select one via Huh.
func discoverAndPick() (string, error) {
	dirs := []string{}

	// 1. XDG config dir.
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		dirs = append(dirs, filepath.Join(xdg, "gearup", "recipes"))
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "gearup", "recipes"))
	}

	// 2. Examples in repo (development convenience).
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "examples", "recipes"))
	}

	entries, err := ui.DiscoverRecipes(dirs)
	if err != nil {
		return "", err
	}
	picked, err := ui.PickRecipe(entries)
	if err != nil {
		return "", err
	}
	return picked.Path, nil
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print gearup version",
		Run: func(*cobra.Command, []string) {
			fmt.Printf("gearup %s\n", version)
		},
	}
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

type stdPrinter struct{}

func (stdPrinter) Printf(format string, args ...any) { fmt.Printf(format, args...) }
func (stdPrinter) Write(p []byte) (int, error)       { return os.Stdout.Write(p) }
```

- [ ] **Step 2: Build + check**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
./gearup version
```

Expected: `gearup 0.0.6-phase4b`.

- [ ] **Step 3: Test suite**

```bash
go test ./...
```

All packages pass.

- [ ] **Step 4: Spot checks**

```bash
# Plan (existing) — should use Lip Gloss preview now:
./gearup plan --recipe ./examples/recipes/frontend.yaml

# Run with --yes (skip Huh confirm on elevation):
./gearup run --recipe ./examples/recipes/frontend.yaml --yes
```

- [ ] **Step 5: Commit**

```bash
git add cmd/gearup/main.go
git commit -m "feat(cli): wire recipe picker, step printer, elevation suppression; bump to 0.0.6-phase4b"
```

---

## Task 7: README update

**File:** `README.md`

- [ ] **Step 1: Update status line**

Find:
```
Status: Phase 4A — in development. Silent check commands (logged, not streamed), `--dry-run` / `gearup plan`, `--yes` for scripted use, per-run log file at `$XDG_STATE_HOME/gearup/logs/`.
```

Replace with:
```
Status: Phase 4B — in development. Interactive recipe picker, styled plan preview, step-level progress rendering, smart elevation-banner suppression.
```

- [ ] **Step 2: Update the Usage section opening**

Find the paragraph starting with:
```
    gearup run --recipe ./examples/recipes/backend.yaml
```

And the paragraph below it starting with "Requires macOS...". Replace both with:

```
    gearup run

Discovers recipes in `$XDG_CONFIG_HOME/gearup/recipes/` and `./examples/recipes/`,
and prompts you to pick one interactively. Or specify directly:

    gearup run --recipe ./examples/recipes/backend.yaml
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document Phase 4B features (picker, preview, spinner)"
```

---

## Task 8: Manual verification (user)

- [ ] **Step 1: Build**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
./gearup version
```

Expected: `gearup 0.0.6-phase4b`.

- [ ] **Step 2: Recipe picker (the headline feature)**

```bash
cd /Users/dlo/Dev/gearup
./gearup run
```

Expected: Huh select appears with "Backend" and "Frontend" as choices. Pick one and see the run proceed.

- [ ] **Step 3: Plan preview**

```bash
./gearup plan --recipe ./examples/recipes/backend.yaml
```

Expected: styled preview with `✓` for installed, `→` for would-install, summary footer with counts.

- [ ] **Step 4: Step printer**

```bash
./gearup run --recipe ./examples/recipes/backend.yaml --yes
```

Expected: each step renders as a single styled line: `✓ [N/12] Step Name  already installed`. Dimmed, compact.

- [ ] **Step 5: Elevation banner suppression**

Since all elevation-required steps are already installed on your machine, the run should NOT show the elevation banner. Confirm: no yellow banner, no Huh confirm prompt.

- [ ] **Step 6: Exit code checks**

```bash
./gearup plan --recipe ./examples/recipes/backend.yaml; echo "exit=$?"
# Expected: exit=0 (everything installed)

./gearup run --recipe ./examples/recipes/frontend.yaml; echo "exit=$?"
# Expected: exit=0
```

---

## Phase 4B completion criteria

- [ ] `go test ./...` passes across all packages.
- [ ] `gearup version` prints `gearup 0.0.6-phase4b`.
- [ ] `gearup run` without `--recipe` shows a Huh picker.
- [ ] `gearup plan` shows a Lip Gloss styled preview.
- [ ] `gearup run` renders steps with `✓` / `→` icons, not raw `[N/12] name: checking...` lines.
- [ ] Elevation banner is suppressed when no elevation step would actually fire.
- [ ] No proprietary content anywhere.

---

## Self-review

**Spec coverage:**
- ✅ Huh recipe picker — Tasks 2, 6
- ✅ Lip Gloss plan preview — Tasks 3, 5, 6
- ✅ Step printer (static, not full Bubbles animation) — Tasks 4, 5, 6
- ✅ Elevation banner suppression — Task 5
- ✅ Version bump + README — Tasks 6, 7

**Type consistency:** `ui.RecipeEntry`, `ui.StepStatus`, `ui.StepPrinter`, `ui.DiscoverRecipes`, `ui.PickRecipe`, `ui.RenderPreview` — all used consistently.

**Placeholder scan:** None.

**Design note on spinner:** The Phase 4B StepPrinter uses ANSI escape codes (`\033[1A\033[2K`) to overwrite the previous line, giving a "single-line-per-step" feel without a full Bubble Tea runtime. This is a pragmatic middle ground — actual per-frame Bubbles animation (option C2 from brainstorming) could be layered on later if the static approach feels insufficient, but it covers the core UX goal of "one clean line per step instead of three noisy lines."
