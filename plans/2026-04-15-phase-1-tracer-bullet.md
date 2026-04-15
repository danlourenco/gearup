# gearup Phase 1 — Tracer Bullet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the thinnest end-to-end slice of gearup: `gearup run --profile <path>` loads one profile file, resolves one recipe from one local path source, and executes one `brew` step with idempotent re-run behavior.

**Architecture:** Five small Go packages (`config`, `exec`, `installer`, `profile`, `runner`) behind a minimal cobra CLI. Every layer gets a focused unit test, plus one integration test that wires real config files to a fake exec runner. Phase 1 proves the full pipeline — config load → recipe resolution → step flattening → installer interface → exec wrapper → check → install → idempotent re-run — against a real brew installation at the end.

**Tech Stack:** Go 1.22+, cobra (CLI), `gopkg.in/yaml.v3` (YAML parsing), Go standard library (`os/exec`, `testing`). Koanf, Charm libraries, and multi-format config are deliberately deferred to later phases.

**Spec reference:** `docs/superpowers/specs/2026-04-15-gearup-design.md` §5 Phase 1 + §6.1 v1 scope.

---

## Scope of this plan

**In Phase 1:**
- `gearup run --profile <path>` subcommand
- `gearup version` subcommand
- Profile YAML parsing: `version`, `name`, `description`, `platform`, `recipe_sources` (local `path:` only), `includes`, inline `steps`
- Recipe YAML parsing: `version`, `name`, `description`, `steps`
- Recipe resolution via a search path built from `recipe_sources`
- Step type: `brew` only (with auto-derived `check` via `brew list --formula`)
- Runner that checks → installs → skips already-installed steps
- Platform guard: refuses to run on non-macOS with exit code 4
- Text output (no styling, no spinners, no Huh forms)

**Deliberately out of Phase 1** (each lives in its own future plan):
- Step types: `curl-pipe-sh`, `shell`, `brew-cask`, `download-binary`, `git-clone`
- Elevation flow (banner, duration, confirmation)
- `post_install` glue
- Huh profile picker, Lip Gloss styling, Bubbles progress, `--dry-run`
- Dedup across includes, transitive `requires:`, git-backed `recipe_sources`
- `gearup list`, `gearup show`, `gearup init`, `gearup doctor`
- Koanf, TOML/JSONC config
- Log files, env-var overrides

---

## File Structure

```
gearup/
├── cmd/gearup/
│   └── main.go                        # cobra root + run + version commands
├── internal/
│   ├── config/
│   │   ├── config.go                  # Profile, Recipe, Step, ResolvedPlan structs
│   │   └── config_test.go             # YAML round-trip tests
│   ├── exec/
│   │   ├── exec.go                    # Runner interface + ShellRunner impl
│   │   ├── fake.go                    # FakeRunner for tests
│   │   └── exec_test.go               # real exec against `true`/`false`/`echo`
│   ├── installer/
│   │   ├── installer.go               # Installer interface + Registry
│   │   └── brew/
│   │       ├── brew.go                # brew Check + Install
│   │       └── brew_test.go           # uses exec.FakeRunner
│   ├── profile/
│   │   ├── profile.go                 # LoadProfile, LoadRecipe, Resolve
│   │   └── profile_test.go            # tempdir fixtures
│   └── runner/
│       ├── runner.go                  # Runner.Run(plan) orchestration
│       └── runner_test.go             # fake installers
├── testdata/
│   └── phase1/
│       ├── profiles/
│       │   └── example.yaml           # test fixture profile
│       └── recipes/
│           └── example-recipe.yaml    # test fixture recipe
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

### Responsibility boundaries

- **`config`** owns struct shapes and YAML tags. No I/O, no resolution logic, no execution. Pure types.
- **`exec`** wraps `os/exec` with a `Runner` interface so every installer becomes unit-testable via a fake runner. This is the single most important seam in Phase 1 — get it right and everything downstream is cheap to test.
- **`installer`** defines the `Installer` interface and a `Registry` mapping step types to implementations. Sub-packages implement per-type logic.
- **`profile`** loads profile and recipe YAML, resolves recipe references via the configured search path, and flattens into a `config.ResolvedPlan`.
- **`runner`** walks a `ResolvedPlan`, dispatching each step to its installer via the registry. Emits text output via a small `Writer` interface so tests can capture it.
- **`cmd/gearup`** wires everything together. Contains no business logic.

---

## Task 1: Scaffold the repo

**Files:**
- Create: `/Users/dlo/Dev/gearup/go.mod`
- Create: `/Users/dlo/Dev/gearup/.gitignore`
- Create: `/Users/dlo/Dev/gearup/cmd/gearup/main.go`
- Create: `/Users/dlo/Dev/gearup/README.md`

- [ ] **Step 1: Initialize go module**

```bash
cd /Users/dlo/Dev/gearup
go mod init gearup
```

Expected: creates `go.mod` with `module gearup` and a Go version line.

- [ ] **Step 2: Write `.gitignore`**

Create `/Users/dlo/Dev/gearup/.gitignore`:

```
# Binaries
/gearup
/bin/

# Go build artifacts
*.test
*.out

# macOS
.DS_Store

# Editor
.vscode/
.idea/
*.swp
```

- [ ] **Step 3: Write a minimal `main.go` that compiles**

Create `/Users/dlo/Dev/gearup/cmd/gearup/main.go`:

```go
package main

import "fmt"

func main() {
	fmt.Println("gearup (phase 1 scaffold)")
}
```

- [ ] **Step 4: Add cobra and yaml.v3 dependencies**

```bash
cd /Users/dlo/Dev/gearup
go get github.com/spf13/cobra@latest
go get gopkg.in/yaml.v3@latest
```

Expected: `go.mod` and `go.sum` updated.

- [ ] **Step 5: Verify the scaffold builds and runs**

```bash
cd /Users/dlo/Dev/gearup
go build ./...
./gearup
```

Expected: `gearup (phase 1 scaffold)` printed. Binary lives at `./gearup` (in repo root because `cmd/gearup` directory name is chosen as binary name by `go build ./...`? Actually `go build ./...` builds all packages; the binary for `cmd/gearup` lands at `./cmd/gearup/gearup`. To get `./gearup`, use `go build ./cmd/gearup`).

Adjusted command:

```bash
go build -o gearup ./cmd/gearup
./gearup
```

Expected output: `gearup (phase 1 scaffold)`.

- [ ] **Step 6: Write a minimal README**

Create `/Users/dlo/Dev/gearup/README.md`:

```markdown
# gearup

Opinionated, open-source macOS developer-machine bootstrap CLI.

Status: Phase 1 (tracer bullet) — in development.

See `docs/superpowers/specs/2026-04-15-gearup-design.md` for the design.

## Phase 1 usage

    gearup run --profile ./examples/profiles/example.yaml

Requires macOS with Homebrew installed. Only the `brew` step type is supported in Phase 1.
```

- [ ] **Step 7: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add go.mod go.sum .gitignore cmd/gearup/main.go README.md
git commit -m "chore: scaffold gearup repo with cobra and yaml.v3"
```

---

## Task 2: Config schema types

**Files:**
- Create: `/Users/dlo/Dev/gearup/internal/config/config.go`
- Create: `/Users/dlo/Dev/gearup/internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `/Users/dlo/Dev/gearup/internal/config/config_test.go`:

```go
package config_test

import (
	"testing"

	"gopkg.in/yaml.v3"

	"gearup/internal/config"
)

func TestProfileUnmarshal(t *testing.T) {
	const src = `
version: 1
name: "Example Profile"
description: "for tests"
platform:
  os: [darwin]
  arch: [arm64, amd64]
recipe_sources:
  - path: ~/src/my-recipes
includes:
  - recipe: example-recipe
steps:
  - name: "Inline step"
    type: brew
    formula: jq
`
	var p config.Profile
	if err := yaml.Unmarshal([]byte(src), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Version != 1 {
		t.Errorf("Version = %d, want 1", p.Version)
	}
	if p.Name != "Example Profile" {
		t.Errorf("Name = %q, want %q", p.Name, "Example Profile")
	}
	if got := p.Platform.OS; len(got) != 1 || got[0] != "darwin" {
		t.Errorf("Platform.OS = %v, want [darwin]", got)
	}
	if len(p.RecipeSources) != 1 || p.RecipeSources[0].Path != "~/src/my-recipes" {
		t.Errorf("RecipeSources = %+v", p.RecipeSources)
	}
	if len(p.Includes) != 1 || p.Includes[0].Recipe != "example-recipe" {
		t.Errorf("Includes = %+v", p.Includes)
	}
	if len(p.Steps) != 1 || p.Steps[0].Type != "brew" || p.Steps[0].Formula != "jq" {
		t.Errorf("Steps = %+v", p.Steps)
	}
}

func TestRecipeUnmarshal(t *testing.T) {
	const src = `
version: 1
name: example-recipe
description: "test recipe"
steps:
  - name: "Install jq"
    type: brew
    formula: jq
  - name: "Install git"
    type: brew
    formula: git
`
	var r config.Recipe
	if err := yaml.Unmarshal([]byte(src), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Name != "example-recipe" {
		t.Errorf("Name = %q", r.Name)
	}
	if len(r.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(r.Steps))
	}
	if r.Steps[0].Formula != "jq" || r.Steps[1].Formula != "git" {
		t.Errorf("Steps = %+v", r.Steps)
	}
}
```

- [ ] **Step 2: Run the test; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/config/...
```

Expected: package build failure because `internal/config/config.go` doesn't exist yet.

- [ ] **Step 3: Write the minimal implementation**

Create `/Users/dlo/Dev/gearup/internal/config/config.go`:

```go
// Package config defines the data types for gearup profiles and recipes.
// It contains no I/O or resolution logic — just struct shapes and YAML tags.
package config

// Profile is the top-level entry point loaded from a profile YAML file.
type Profile struct {
	Version       int            `yaml:"version"`
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description,omitempty"`
	Platform      Platform       `yaml:"platform,omitempty"`
	RecipeSources []RecipeSource `yaml:"recipe_sources,omitempty"`
	Includes      []Include      `yaml:"includes,omitempty"`
	Steps         []Step         `yaml:"steps,omitempty"`
}

// Platform constrains which OS/arch a profile (or step) applies to.
type Platform struct {
	OS   []string `yaml:"os,omitempty"`
	Arch []string `yaml:"arch,omitempty"`
}

// RecipeSource declares a location where recipes can be resolved from.
// Phase 1 supports only local filesystem paths.
type RecipeSource struct {
	Path string `yaml:"path,omitempty"`
}

// Include references a recipe by name; resolved against the search path.
type Include struct {
	Recipe string `yaml:"recipe"`
}

// Recipe is a bundle of steps loaded from a recipe YAML file.
type Recipe struct {
	Version     int    `yaml:"version"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Steps       []Step `yaml:"steps"`
}

// Step is one unit of provisioning work.
// Phase 1 only implements type: brew, so the type-specific fields here
// are minimal; future phases add cask, url, repo, etc.
type Step struct {
	Name              string   `yaml:"name"`
	Type              string   `yaml:"type"`
	Check             string   `yaml:"check,omitempty"`
	RequiresElevation bool     `yaml:"requires_elevation,omitempty"`
	Platform          Platform `yaml:"platform,omitempty"`

	// brew-specific
	Formula string `yaml:"formula,omitempty"`
}

// ResolvedPlan is a flattened, ordered list of steps produced by
// resolving a profile's recipe references.
type ResolvedPlan struct {
	Profile *Profile
	Steps   []Step
}
```

- [ ] **Step 4: Run the tests; expect PASS**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/config/...
```

Expected: `ok  	gearup/internal/config`.

- [ ] **Step 5: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add internal/config/
git commit -m "feat(config): add Profile, Recipe, Step, ResolvedPlan types"
```

---

## Task 3: Exec wrapper with a fake runner

**Files:**
- Create: `/Users/dlo/Dev/gearup/internal/exec/exec.go`
- Create: `/Users/dlo/Dev/gearup/internal/exec/fake.go`
- Create: `/Users/dlo/Dev/gearup/internal/exec/exec_test.go`

- [ ] **Step 1: Write the failing test**

Create `/Users/dlo/Dev/gearup/internal/exec/exec_test.go`:

```go
package exec_test

import (
	"context"
	"strings"
	"testing"

	gearexec "gearup/internal/exec"
)

func TestShellRunner_TrueExits0(t *testing.T) {
	r := &gearexec.ShellRunner{}
	res, err := r.Run(context.Background(), "true")
	if err != nil {
		t.Fatalf("unexpected spawn error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestShellRunner_FalseExitsNonzero(t *testing.T) {
	r := &gearexec.ShellRunner{}
	res, err := r.Run(context.Background(), "false")
	if err != nil {
		t.Fatalf("unexpected spawn error: %v", err)
	}
	if res.ExitCode == 0 {
		t.Error("ExitCode = 0, want non-zero")
	}
}

func TestShellRunner_CapturesStdout(t *testing.T) {
	r := &gearexec.ShellRunner{}
	res, err := r.Run(context.Background(), "echo hello-gearup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello-gearup") {
		t.Errorf("Stdout = %q, want contains hello-gearup", res.Stdout)
	}
}

func TestFakeRunner_ReturnsProgrammedResult(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 0, Stdout: "/opt/homebrew/Cellar/jq"}, nil)
	res, err := f.Run(context.Background(), "brew list --formula jq")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "Cellar/jq") {
		t.Errorf("Stdout = %q", res.Stdout)
	}
	if got := f.Calls(); len(got) != 1 || got[0] != "brew list --formula jq" {
		t.Errorf("Calls = %v", got)
	}
}

func TestFakeRunner_UnstubbedCommandFails(t *testing.T) {
	f := gearexec.NewFakeRunner()
	_, err := f.Run(context.Background(), "some unstubbed command")
	if err == nil {
		t.Error("want error for unstubbed command, got nil")
	}
}
```

- [ ] **Step 2: Run the test; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/exec/...
```

Expected: build failure, package doesn't exist.

- [ ] **Step 3: Write `exec.go`**

Create `/Users/dlo/Dev/gearup/internal/exec/exec.go`:

```go
// Package exec wraps os/exec with a Runner interface that installers depend
// on. The interface makes every installer unit-testable via FakeRunner.
package exec

import (
	"bytes"
	"context"
	"io"
	osexec "os/exec"
)

// Result is the outcome of running a command.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Runner runs a shell command string and returns its Result.
// A non-zero exit code is reported in Result.ExitCode and is NOT an error;
// err is non-nil only if the command could not be spawned at all.
type Runner interface {
	Run(ctx context.Context, cmd string) (Result, error)
}

// ShellRunner runs commands via `bash -c`.
// Optional Stdout/Stderr writers receive a live stream of the command's output
// (multiplexed with the captured buffers in the returned Result).
type ShellRunner struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Run implements Runner.
func (r *ShellRunner) Run(ctx context.Context, cmd string) (Result, error) {
	c := osexec.CommandContext(ctx, "bash", "-c", cmd)
	var outBuf, errBuf bytes.Buffer
	if r.Stdout != nil {
		c.Stdout = io.MultiWriter(&outBuf, r.Stdout)
	} else {
		c.Stdout = &outBuf
	}
	if r.Stderr != nil {
		c.Stderr = io.MultiWriter(&errBuf, r.Stderr)
	} else {
		c.Stderr = &errBuf
	}
	err := c.Run()
	res := Result{Stdout: outBuf.String(), Stderr: errBuf.String()}
	if exitErr, ok := err.(*osexec.ExitError); ok {
		res.ExitCode = exitErr.ExitCode()
		return res, nil
	}
	if err != nil {
		return res, err
	}
	res.ExitCode = 0
	return res, nil
}
```

- [ ] **Step 4: Write `fake.go`**

Create `/Users/dlo/Dev/gearup/internal/exec/fake.go`:

```go
package exec

import (
	"context"
	"fmt"
	"sync"
)

// FakeRunner is a test double for Runner. Register expected commands with
// On().Return(...); any call to an unstubbed command returns an error.
type FakeRunner struct {
	mu    sync.Mutex
	stubs map[string]fakeResponse
	calls []string
}

type fakeResponse struct {
	result Result
	err    error
}

// NewFakeRunner returns an empty FakeRunner.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{stubs: map[string]fakeResponse{}}
}

// On starts a stub registration for the given command string.
type stubBuilder struct {
	f   *FakeRunner
	cmd string
}

// On registers a stub for an exact command string.
func (f *FakeRunner) On(cmd string) *stubBuilder {
	return &stubBuilder{f: f, cmd: cmd}
}

// Return sets the Result and error returned when the stubbed command runs.
func (b *stubBuilder) Return(res Result, err error) {
	b.f.mu.Lock()
	defer b.f.mu.Unlock()
	b.f.stubs[b.cmd] = fakeResponse{result: res, err: err}
}

// Run implements Runner.
func (f *FakeRunner) Run(_ context.Context, cmd string) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, cmd)
	resp, ok := f.stubs[cmd]
	if !ok {
		return Result{}, fmt.Errorf("FakeRunner: unstubbed command %q", cmd)
	}
	return resp.result, resp.err
}

// Calls returns the commands that have been invoked, in order.
func (f *FakeRunner) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}
```

- [ ] **Step 5: Run the tests; expect PASS**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/exec/... -v
```

Expected: all five tests pass. `bash` must be on PATH (it is, on macOS).

- [ ] **Step 6: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add internal/exec/
git commit -m "feat(exec): add Runner interface, ShellRunner, and FakeRunner"
```

---

## Task 4: Installer interface and registry

**Files:**
- Create: `/Users/dlo/Dev/gearup/internal/installer/installer.go`
- Create: `/Users/dlo/Dev/gearup/internal/installer/installer_test.go`

- [ ] **Step 1: Write the failing test**

Create `/Users/dlo/Dev/gearup/internal/installer/installer_test.go`:

```go
package installer_test

import (
	"context"
	"testing"

	"gearup/internal/config"
	"gearup/internal/installer"
)

type stubInstaller struct {
	name string
}

func (s *stubInstaller) Check(_ context.Context, _ config.Step) (bool, error) { return true, nil }
func (s *stubInstaller) Install(_ context.Context, _ config.Step) error       { return nil }

func TestRegistry_GetKnownType(t *testing.T) {
	reg := installer.Registry{"brew": &stubInstaller{name: "brew"}}
	got, err := reg.Get("brew")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("want non-nil Installer")
	}
}

func TestRegistry_GetUnknownType(t *testing.T) {
	reg := installer.Registry{}
	_, err := reg.Get("not-a-type")
	if err == nil {
		t.Error("want error for unknown step type, got nil")
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/installer/...
```

Expected: build failure.

- [ ] **Step 3: Write the implementation**

Create `/Users/dlo/Dev/gearup/internal/installer/installer.go`:

```go
// Package installer defines the Installer interface and a Registry mapping
// step types to implementations. Type-specific installers live in sub-packages.
package installer

import (
	"context"
	"fmt"

	"gearup/internal/config"
)

// Installer provides idempotent check and install for one step type.
//
// Check reports whether the step is already satisfied on the host.
// When Check returns (true, nil) the runner skips Install for that step.
//
// Install executes the provisioning action. It must return an error if
// the action failed for any reason.
type Installer interface {
	Check(ctx context.Context, step config.Step) (installed bool, err error)
	Install(ctx context.Context, step config.Step) error
}

// Registry maps step type names (e.g. "brew") to their Installer.
type Registry map[string]Installer

// Get returns the Installer for a step type or an error if none is registered.
func (r Registry) Get(stepType string) (Installer, error) {
	inst, ok := r[stepType]
	if !ok {
		return nil, fmt.Errorf("no installer registered for step type %q", stepType)
	}
	return inst, nil
}
```

- [ ] **Step 4: Run; expect PASS**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/installer/...
```

Expected: `ok  	gearup/internal/installer`.

- [ ] **Step 5: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add internal/installer/
git commit -m "feat(installer): add Installer interface and Registry"
```

---

## Task 5: Brew installer

**Files:**
- Create: `/Users/dlo/Dev/gearup/internal/installer/brew/brew.go`
- Create: `/Users/dlo/Dev/gearup/internal/installer/brew/brew_test.go`

- [ ] **Step 1: Write the failing test**

Create `/Users/dlo/Dev/gearup/internal/installer/brew/brew_test.go`:

```go
package brew_test

import (
	"context"
	"testing"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer/brew"
)

func TestBrew_CheckInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 0, Stdout: "/opt/homebrew/Cellar/jq/1.7"}, nil)

	inst := &brew.Installer{Runner: f}
	ok, err := inst.Check(context.Background(), config.Step{Name: "jq", Type: "brew", Formula: "jq"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true")
	}
}

func TestBrew_CheckNotInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 1, Stderr: "Error: No such keg"}, nil)

	inst := &brew.Installer{Runner: f}
	ok, err := inst.Check(context.Background(), config.Step{Name: "jq", Type: "brew", Formula: "jq"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("installed = true, want false")
	}
}

func TestBrew_CheckMissingFormula(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &brew.Installer{Runner: f}
	_, err := inst.Check(context.Background(), config.Step{Name: "bad", Type: "brew"})
	if err == nil {
		t.Error("want error for missing formula, got nil")
	}
}

func TestBrew_InstallSuccess(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew install jq").Return(gearexec.Result{ExitCode: 0, Stdout: "==> Pouring jq"}, nil)

	inst := &brew.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{Name: "jq", Type: "brew", Formula: "jq"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBrew_InstallFailure(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew install missing-pkg").Return(gearexec.Result{ExitCode: 1, Stderr: "Error: No available formula"}, nil)

	inst := &brew.Installer{Runner: f}
	err := inst.Install(context.Background(), config.Step{Name: "missing", Type: "brew", Formula: "missing-pkg"})
	if err == nil {
		t.Error("want error for failed install, got nil")
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/installer/brew/...
```

- [ ] **Step 3: Write the implementation**

Create `/Users/dlo/Dev/gearup/internal/installer/brew/brew.go`:

```go
// Package brew implements the "brew" step type backed by Homebrew.
package brew

import (
	"context"
	"fmt"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
)

// Installer is the brew step installer.
type Installer struct {
	Runner gearexec.Runner
}

// Check runs `brew list --formula <name>`; exit 0 means installed.
func (i *Installer) Check(ctx context.Context, step config.Step) (bool, error) {
	if step.Formula == "" {
		return false, fmt.Errorf("brew step %q missing formula", step.Name)
	}
	cmd := fmt.Sprintf("brew list --formula %s", step.Formula)
	res, err := i.Runner.Run(ctx, cmd)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

// Install runs `brew install <formula>` and fails on non-zero exit.
func (i *Installer) Install(ctx context.Context, step config.Step) error {
	if step.Formula == "" {
		return fmt.Errorf("brew step %q missing formula", step.Name)
	}
	cmd := fmt.Sprintf("brew install %s", step.Formula)
	res, err := i.Runner.Run(ctx, cmd)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("brew install %s failed (exit %d): %s", step.Formula, res.ExitCode, res.Stderr)
	}
	return nil
}
```

- [ ] **Step 4: Run; expect PASS**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/installer/brew/... -v
```

Expected: all five tests pass.

- [ ] **Step 5: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add internal/installer/brew/
git commit -m "feat(installer/brew): implement brew Check and Install"
```

---

## Task 6: Profile and recipe loading + resolution

**Files:**
- Create: `/Users/dlo/Dev/gearup/internal/profile/profile.go`
- Create: `/Users/dlo/Dev/gearup/internal/profile/profile_test.go`

- [ ] **Step 1: Write the failing test**

Create `/Users/dlo/Dev/gearup/internal/profile/profile_test.go`:

```go
package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"gearup/internal/profile"
)

// writeFile is a helper for fixture setup.
func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestLoadProfile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "profile.yaml", `
version: 1
name: "Test"
includes:
  - recipe: sample
`)
	p, err := profile.LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if p.Name != "Test" {
		t.Errorf("Name = %q", p.Name)
	}
	if len(p.Includes) != 1 || p.Includes[0].Recipe != "sample" {
		t.Errorf("Includes = %+v", p.Includes)
	}
}

func TestResolve_RecipeFromLocalPath(t *testing.T) {
	root := t.TempDir()
	recipesDir := filepath.Join(root, "my-recipes")

	writeFile(t, recipesDir, "sample.yaml", `
version: 1
name: sample
steps:
  - name: "Install jq"
    type: brew
    formula: jq
`)
	profilePath := writeFile(t, root, "profile.yaml", `
version: 1
name: "Test"
recipe_sources:
  - path: `+recipesDir+`
includes:
  - recipe: sample
steps:
  - name: "Inline step"
    type: brew
    formula: git
`)

	p, err := profile.LoadProfile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	plan, err := profile.Resolve(p, filepath.Dir(profilePath))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got, want := len(plan.Steps), 2; got != want {
		t.Fatalf("Steps len = %d, want %d (%+v)", got, want, plan.Steps)
	}
	if plan.Steps[0].Formula != "jq" {
		t.Errorf("Steps[0].Formula = %q, want jq", plan.Steps[0].Formula)
	}
	if plan.Steps[1].Formula != "git" {
		t.Errorf("Steps[1].Formula = %q, want git", plan.Steps[1].Formula)
	}
}

func TestResolve_RecipeNotFound(t *testing.T) {
	root := t.TempDir()
	profilePath := writeFile(t, root, "profile.yaml", `
version: 1
name: "Test"
includes:
  - recipe: does-not-exist
`)
	p, err := profile.LoadProfile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	_, err = profile.Resolve(p, filepath.Dir(profilePath))
	if err == nil {
		t.Error("want error for missing recipe, got nil")
	}
}

func TestResolve_RelativePathResolvedAgainstProfileDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "recipes"), "sample.yaml", `
version: 1
name: sample
steps:
  - name: s
    type: brew
    formula: jq
`)
	profilePath := writeFile(t, root, "profile.yaml", `
version: 1
name: "Test"
recipe_sources:
  - path: ./recipes
includes:
  - recipe: sample
`)
	p, err := profile.LoadProfile(profilePath)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	plan, err := profile.Resolve(p, filepath.Dir(profilePath))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Formula != "jq" {
		t.Errorf("Steps = %+v", plan.Steps)
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/profile/...
```

- [ ] **Step 3: Write the implementation**

Create `/Users/dlo/Dev/gearup/internal/profile/profile.go`:

```go
// Package profile loads profile and recipe YAML files and flattens them
// into a ResolvedPlan. Phase 1: local path sources only, no transitive
// requires, no deduplication.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"gearup/internal/config"
)

// LoadProfile reads and parses a profile YAML file.
func LoadProfile(path string) (*config.Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile %s: %w", path, err)
	}
	var p config.Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile %s: %w", path, err)
	}
	return &p, nil
}

// LoadRecipe reads and parses a recipe YAML file.
func LoadRecipe(path string) (*config.Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read recipe %s: %w", path, err)
	}
	var r config.Recipe
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse recipe %s: %w", path, err)
	}
	return &r, nil
}

// Resolve walks profile.Includes and inline Steps, producing a ResolvedPlan.
//
// profileDir is the directory of the profile file; relative recipe_source
// paths are resolved against it.
func Resolve(p *config.Profile, profileDir string) (*config.ResolvedPlan, error) {
	searchPath := buildSearchPath(p, profileDir)

	var steps []config.Step
	for _, inc := range p.Includes {
		recipePath, err := findRecipe(inc.Recipe, searchPath)
		if err != nil {
			return nil, err
		}
		r, err := LoadRecipe(recipePath)
		if err != nil {
			return nil, err
		}
		steps = append(steps, r.Steps...)
	}
	steps = append(steps, p.Steps...)
	return &config.ResolvedPlan{Profile: p, Steps: steps}, nil
}

func buildSearchPath(p *config.Profile, profileDir string) []string {
	var paths []string
	for _, src := range p.RecipeSources {
		if src.Path == "" {
			continue
		}
		expanded := expandHome(src.Path)
		if !filepath.IsAbs(expanded) {
			expanded = filepath.Join(profileDir, expanded)
		}
		paths = append(paths, expanded)
	}
	return paths
}

func findRecipe(name string, searchPath []string) (string, error) {
	for _, dir := range searchPath {
		candidate := filepath.Join(dir, name+".yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("recipe %q not found in any recipe source", name)
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
```

- [ ] **Step 4: Run; expect PASS**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/profile/... -v
```

Expected: all four tests pass.

- [ ] **Step 5: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add internal/profile/
git commit -m "feat(profile): load profile/recipe and resolve into flat plan"
```

---

## Task 7: Runner orchestration

**Files:**
- Create: `/Users/dlo/Dev/gearup/internal/runner/runner.go`
- Create: `/Users/dlo/Dev/gearup/internal/runner/runner_test.go`

- [ ] **Step 1: Write the failing test**

Create `/Users/dlo/Dev/gearup/internal/runner/runner_test.go`:

```go
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
```

- [ ] **Step 2: Run; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/runner/...
```

- [ ] **Step 3: Write the implementation**

Create `/Users/dlo/Dev/gearup/internal/runner/runner.go`:

```go
// Package runner executes a ResolvedPlan by dispatching each step to its
// registered installer. It stops on the first error (fail-fast).
package runner

import (
	"context"
	"fmt"

	"gearup/internal/config"
	"gearup/internal/installer"
)

// Writer is the minimal output surface the Runner needs. Real use passes
// a stdout wrapper; tests pass a buffer.
type Writer interface {
	Printf(format string, args ...any)
}

// Runner orchestrates plan execution.
type Runner struct {
	Registry installer.Registry
	Out      Writer
}

// Run walks plan.Steps in order. For each step: look up its installer,
// call Check; if already installed, skip; otherwise call Install.
// Returns on the first error.
func (r *Runner) Run(ctx context.Context, plan *config.ResolvedPlan) error {
	total := len(plan.Steps)
	for i, step := range plan.Steps {
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
			continue
		}
		r.Out.Printf("[%d/%d] %s: installing...\n", idx, total, step.Name)
		if err := inst.Install(ctx, step); err != nil {
			return fmt.Errorf("step %d (%s) install failed: %w", idx, step.Name, err)
		}
		r.Out.Printf("[%d/%d] %s: installed\n", idx, total, step.Name)
	}
	return nil
}
```

- [ ] **Step 4: Run; expect PASS**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/runner/... -v
```

Expected: all five tests pass.

- [ ] **Step 5: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add internal/runner/
git commit -m "feat(runner): add fail-fast plan orchestration"
```

---

## Task 8: CLI wiring (`gearup run` + `gearup version`)

**Files:**
- Modify: `/Users/dlo/Dev/gearup/cmd/gearup/main.go` (full rewrite)

- [ ] **Step 1: Rewrite `main.go`**

Replace `/Users/dlo/Dev/gearup/cmd/gearup/main.go` with:

```go
// Command gearup is an open-source macOS developer-machine bootstrap CLI.
// Phase 1: runs a single profile with brew steps only.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	gearexec "gearup/internal/exec"
	"gearup/internal/installer"
	"gearup/internal/installer/brew"
	"gearup/internal/profile"
	"gearup/internal/runner"
)

const version = "0.0.1-phase1"

func main() {
	root := &cobra.Command{
		Use:   "gearup",
		Short: "Open-source macOS developer-machine bootstrap CLI",
	}
	root.AddCommand(runCmd(), versionCmd())
	if err := root.Execute(); err != nil {
		// cobra already printed the error
		os.Exit(1)
	}
}

func runCmd() *cobra.Command {
	var profilePath string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a provisioning profile",
		RunE: func(c *cobra.Command, args []string) error {
			if runtime.GOOS != "darwin" {
				fmt.Fprintln(os.Stderr, "gearup currently supports macOS only")
				os.Exit(4)
			}
			if profilePath == "" {
				return fmt.Errorf("--profile is required (Phase 1)")
			}
			absProfile, err := filepath.Abs(profilePath)
			if err != nil {
				return err
			}
			p, err := profile.LoadProfile(absProfile)
			if err != nil {
				return err
			}
			plan, err := profile.Resolve(p, filepath.Dir(absProfile))
			if err != nil {
				return err
			}

			shell := &gearexec.ShellRunner{Stdout: os.Stdout, Stderr: os.Stderr}
			reg := installer.Registry{
				"brew": &brew.Installer{Runner: shell},
			}
			r := &runner.Runner{Registry: reg, Out: stdPrinter{}}

			fmt.Printf("PROFILE: %s  (%d steps)\n\n", p.Name, len(plan.Steps))
			if err := r.Run(context.Background(), plan); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			fmt.Println("\nDone.")
			return nil
		},
	}
	cmd.Flags().StringVar(&profilePath, "profile", "", "path to profile YAML (required)")
	return cmd
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

// stdPrinter adapts os.Stdout to runner.Writer.
type stdPrinter struct{}

func (stdPrinter) Printf(format string, args ...any) { fmt.Printf(format, args...) }
```

- [ ] **Step 2: Verify it builds**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
```

Expected: `./gearup` binary created, no errors.

- [ ] **Step 3: Run `gearup version`**

```bash
./gearup version
```

Expected: `gearup 0.0.1-phase1`

- [ ] **Step 4: Run `gearup run` without --profile and verify error**

```bash
./gearup run
echo "exit=$?"
```

Expected: an error message about `--profile is required` and a non-zero exit code.

- [ ] **Step 5: Run all tests to make sure nothing regressed**

```bash
cd /Users/dlo/Dev/gearup
go test ./...
```

Expected: all packages pass (`config`, `exec`, `installer`, `installer/brew`, `profile`, `runner`).

- [ ] **Step 6: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add cmd/gearup/main.go
git commit -m "feat(cli): add run and version commands wired to runner"
```

---

## Task 9: Example profile and recipe fixtures

**Files:**
- Create: `/Users/dlo/Dev/gearup/examples/profiles/example.yaml`
- Create: `/Users/dlo/Dev/gearup/examples/recipes/example-recipe.yaml`

- [ ] **Step 1: Create the example recipe**

Create `/Users/dlo/Dev/gearup/examples/recipes/example-recipe.yaml`:

```yaml
version: 1
name: example-recipe
description: "Phase 1 tracer-bullet example: installs jq via brew"

steps:
  - name: "jq (via brew)"
    type: brew
    formula: jq
```

- [ ] **Step 2: Create the example profile**

Create `/Users/dlo/Dev/gearup/examples/profiles/example.yaml`:

```yaml
version: 1
name: "Phase 1 Example"
description: "Minimal profile proving the end-to-end gearup pipeline"

platform:
  os: [darwin]

recipe_sources:
  - path: ../recipes

includes:
  - recipe: example-recipe
```

- [ ] **Step 3: Validate the example parses and resolves via a smoke test**

Append to `/Users/dlo/Dev/gearup/internal/profile/profile_test.go`:

```go
func TestResolve_ExampleFixture(t *testing.T) {
	p, err := profile.LoadProfile("../../examples/profiles/example.yaml")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	plan, err := profile.Resolve(p, "../../examples/profiles")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("Steps len = %d, want 1 (%+v)", len(plan.Steps), plan.Steps)
	}
	if plan.Steps[0].Type != "brew" || plan.Steps[0].Formula != "jq" {
		t.Errorf("Steps[0] = %+v", plan.Steps[0])
	}
}
```

- [ ] **Step 4: Run; expect PASS**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/profile/... -v
```

Expected: the new `TestResolve_ExampleFixture` passes alongside the others.

- [ ] **Step 5: Commit**

```bash
cd /Users/dlo/Dev/gearup
git add examples/ internal/profile/profile_test.go
git commit -m "feat(examples): add phase-1 example profile and recipe"
```

---

## Task 10: End-to-end verification against real Homebrew

This task is a manual smoke test. It proves the full pipeline works against a real `brew` on a real macOS machine. **Do this on your own dev Mac.**

- [ ] **Step 1: Confirm Homebrew is installed**

```bash
command -v brew
```

Expected: a path like `/opt/homebrew/bin/brew`. If not, install Homebrew first via the official installer.

- [ ] **Step 2: Build the binary**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
```

Expected: `./gearup` built with no errors.

- [ ] **Step 3: Ensure the example profile's tool is NOT currently installed**

```bash
brew list --formula jq >/dev/null 2>&1 && echo "INSTALLED" || echo "NOT INSTALLED"
```

If `INSTALLED`, uninstall with `brew uninstall jq` so the tracer test exercises the install path on first run. (If you legitimately use `jq`, edit `examples/recipes/example-recipe.yaml` to use a formula you don't have — e.g., `figlet`.)

- [ ] **Step 4: First run — should install jq**

```bash
cd /Users/dlo/Dev/gearup
./gearup run --profile ./examples/profiles/example.yaml
```

Expected output (approximately):

```
PROFILE: Phase 1 Example  (1 steps)

[1/1] jq (via brew): checking...
[1/1] jq (via brew): installing...
==> Downloading https://ghcr.io/v2/homebrew/core/jq/...
==> Pouring jq--...
[1/1] jq (via brew): installed

Done.
```

And:

```bash
command -v jq && jq --version
```

Expected: a path and a version string, proving jq is installed.

- [ ] **Step 5: Second run — should skip (proves idempotency)**

```bash
cd /Users/dlo/Dev/gearup
./gearup run --profile ./examples/profiles/example.yaml
```

Expected output:

```
PROFILE: Phase 1 Example  (1 steps)

[1/1] jq (via brew): checking...
[1/1] jq (via brew): already installed — skip

Done.
```

No brew commands should have been executed after the check. This is the core idempotency guarantee of Phase 1.

- [ ] **Step 6: Exit-code verification**

```bash
cd /Users/dlo/Dev/gearup
./gearup run --profile ./examples/profiles/example.yaml
echo "exit=$?"
```

Expected: `exit=0`.

- [ ] **Step 7: Platform-guard smoke check (optional)**

If you have access to a Linux machine or a Linux Docker container, cross-compile and confirm the platform guard fires:

```bash
cd /Users/dlo/Dev/gearup
GOOS=linux GOARCH=amd64 go build -o gearup-linux ./cmd/gearup
# Copy to a Linux host, run:
# ./gearup-linux run --profile ...
# Expected: "gearup currently supports macOS only" and exit code 4.
```

Not a blocker for Phase 1 completion; nice-to-have confirmation.

- [ ] **Step 8: Document what you observed in the README**

Append to `/Users/dlo/Dev/gearup/README.md`:

```markdown

## Phase 1 verification

Manual end-to-end test (against real Homebrew):

1. `go build -o gearup ./cmd/gearup`
2. `./gearup run --profile ./examples/profiles/example.yaml` — installs jq
3. `./gearup run --profile ./examples/profiles/example.yaml` — skips jq (idempotent)

All unit tests: `go test ./...`
```

- [ ] **Step 9: Commit the README update and tag the Phase 1 milestone**

```bash
cd /Users/dlo/Dev/gearup
git add README.md
git commit -m "docs: add phase-1 verification steps to README"
git tag -a v0.0.1-phase1 -m "Phase 1 tracer bullet complete"
```

---

## Phase 1 completion criteria

All must be true before the phase is considered done:

- [ ] `go test ./...` passes with zero failures.
- [ ] `gearup version` prints the version string.
- [ ] `gearup run --profile ./examples/profiles/example.yaml` installs the example formula on first run.
- [ ] The same command on a second run skips the already-installed step without invoking `brew install`.
- [ ] The binary refuses to run on non-macOS with exit code 4.
- [ ] No proprietary/internal names or paths appear anywhere in source, docs, examples, tests, commit messages, or output. (Scan before tagging.)

---

## Future phases (not in this plan)

Each gets its own plan document written after Phase 1 lands:

- **Phase 2:** `curl-pipe-sh` and `shell` step types.
- **Phase 3:** elevation flow (`requires_elevation`, `elevation.message`, `duration` countdown, Huh confirmation).
- **Phase 4:** Huh profile picker, Lip Gloss plan preview, Bubbles progress UI, `--dry-run` / `gearup plan`.
- **Phase 5:** `brew-cask`, `download-binary`, `git-clone` step types + `post_install` glue.
- **Phase 6:** Git-backed `recipe_sources` with cache and `ref:` pinning.
- **Phase 7:** Transitive `requires:` between recipes, override precedence policy, dedup semantics.
- **Phase 8+:** `gearup list`, `gearup show`, `gearup init`, `gearup doctor`; Linux support; TOML/JSONC config; log file with rotation; Koanf adoption.

---

## Self-review

**Spec coverage (§6.1 v1 scope — Phase 1 subset):**
- ✅ `gearup run` + `gearup version` — Tasks 8, 10
- ✅ Step type `brew` — Task 5
- ✅ Profile loading from local YAML — Task 6
- ✅ `recipe_sources` with local `path:` entries — Task 6
- ✅ `includes:` with flat recipe resolution — Task 6
- ✅ `check` + idempotent re-run — Tasks 5, 7, 10
- ✅ Bullet-list plan preview — light in Phase 1 (Task 8 prints a header + per-step lines; full preview lives in Phase 4)
- ✅ TTY required — **NOT explicit in Phase 1.** Platform guard is (Task 8). TTY check deferred to Phase 4 when it gates the Huh picker. Phase 1 is CLI-only so TTY gating adds no value yet.
- ✅ Exit codes 0 (happy), 1 (step failed via `os.Exit(1)` on runner error), 4 (platform mismatch) — Task 8. Exit codes 2 (config error) and 3 (user aborted) not yet triggered in Phase 1 because there's no interactive confirmation to abort and config errors bubble up as exit 1. These get dedicated handling in Phase 3/4.
- ✅ macOS only — Task 8 platform guard
- ✅ No company-specific content anywhere — verified in Task 10 Step 9 completion criterion

**Placeholder scan:** No "TBD", "TODO", "similar to", "implement later", or unexplained references. Every task has full code, exact paths, and exact commands.

**Type consistency:** `config.Profile`, `config.Recipe`, `config.Step`, `config.ResolvedPlan`, `installer.Installer`, `installer.Registry`, `exec.Runner`, `exec.Result`, `exec.ShellRunner`, `exec.FakeRunner`, `runner.Runner`, `runner.Writer` — names used identically across every task that references them. Checked.

**Known trade-offs (intentional, deferred):**
- Non-zero exits from `exec.Runner.Run` return `nil` error with a non-zero `Result.ExitCode`. Installers decide how to interpret. Documented in the `Runner` doc comment.
- Phase 1 does not deduplicate steps across includes. First-match-wins dedup lands in Phase 7 alongside transitive `requires:`.
- Phase 1 prints output directly from `ShellRunner` alongside runner status lines. Interleaving may look a bit raw; polished rendering is Phase 4's job.
