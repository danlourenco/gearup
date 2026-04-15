# gearup Phase 2 — `curl-pipe-sh` and `shell` step types + developer recipe

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the `curl-pipe-sh` and `shell` step types so gearup can install tools that don't come from Homebrew (e.g., Homebrew itself, nvm), and ship a user-facing developer recipe that exercises all three step types across the 11-tool macOS dev toolchain.

**Architecture:** Two new installer sub-packages (`internal/installer/curlpipe`, `internal/installer/shell`) implementing the existing `Installer` interface. Add `URL`, `Shell`, `Args`, and `Install` fields to `config.Step`. Wire both new installers into the registry in `cmd/gearup/main.go`. Ship a new `examples/recipes/dev-stack.yaml` recipe with the 11 tools and a matching profile.

**Tech Stack:** Same as Phase 1 (Go, cobra, yaml.v3, stdlib). No new external dependencies.

**Spec reference:** `docs/superpowers/specs/2026-04-15-gearup-design.md` §3.3 (step types), §5 Phase 2.

---

## Scope of this plan

**In Phase 2:**
- New step type `curl-pipe-sh`: runs a remote installer via `curl -fsSL <url> | <shell> [args...]`. Requires explicit `check` (no auto-derived check for this type).
- New step type `shell`: runs arbitrary `install` shell command. Requires explicit `check`.
- New fields on `config.Step`: `URL`, `Shell` (default `bash`), `Args`, `Install`.
- Registry wiring in `cmd/gearup/main.go` so `run` dispatches all three types.
- New developer recipe at `examples/recipes/dev-stack.yaml` with 11 tools:
  - Homebrew (`curl-pipe-sh`)
  - Git, OpenJDK 21, Docker CLI, Docker Compose, Colima, AWS CLI, aws-iam-authenticator, kubectl, jq (`brew`)
  - nvm (`curl-pipe-sh`)
- New profile at `examples/profiles/dev-stack.yaml` that includes the recipe.
- Smoke test that loads and resolves the new dev-stack profile (mirrors the Phase 1 fixture test).

**Deliberately out of Phase 2:**
- Elevation flow (Phase 3).
- `post_install` glue (Phase 5) — any needed post-install commands today go into a separate `shell` step.
- Huh picker, dry-run, styling (Phase 4).
- Transitive `requires:` between recipes (Phase 7).
- PATH bootstrap so newly-installed Homebrew is usable in the same `gearup run` (noted at the end of this plan; not required for users who already have Homebrew).

---

## File structure (changes)

```
gearup/
├── internal/
│   ├── config/
│   │   └── config.go                    # MODIFY: add URL, Shell, Args, Install fields to Step
│   └── installer/
│       ├── curlpipe/
│       │   ├── curlpipe.go              # NEW
│       │   └── curlpipe_test.go         # NEW
│       └── shell/
│           ├── shell.go                 # NEW
│           └── shell_test.go            # NEW
├── cmd/gearup/
│   └── main.go                          # MODIFY: register curlpipe + shell in the Registry
└── examples/
    ├── profiles/
    │   └── dev-stack.yaml               # NEW
    └── recipes/
        └── dev-stack.yaml               # NEW
```

---

## Task 1: Extend the `config.Step` type

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Extend the `TestProfileUnmarshal` test to cover the new fields**

Append to `/Users/dlo/Dev/gearup/internal/config/config_test.go` (new test function at end of file):

```go
func TestStepUnmarshal_CurlPipeAndShell(t *testing.T) {
	const src = `
version: 1
name: "Mixed"
steps:
  - name: "nvm"
    type: curl-pipe-sh
    url: https://example.com/install.sh
    shell: bash
    args: ["--quiet"]
    check: 'test -f "$HOME/.nvm/nvm.sh"'
  - name: "Custom"
    type: shell
    check: command -v thing
    install: |
      echo "installing thing"
      touch thing
`
	var p config.Profile
	if err := yaml.Unmarshal([]byte(src), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(p.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(p.Steps))
	}
	cp := p.Steps[0]
	if cp.Type != "curl-pipe-sh" || cp.URL != "https://example.com/install.sh" {
		t.Errorf("curl-pipe step = %+v", cp)
	}
	if cp.Shell != "bash" {
		t.Errorf("Shell = %q, want bash", cp.Shell)
	}
	if len(cp.Args) != 1 || cp.Args[0] != "--quiet" {
		t.Errorf("Args = %v", cp.Args)
	}
	sh := p.Steps[1]
	if sh.Type != "shell" || sh.Install == "" {
		t.Errorf("shell step = %+v", sh)
	}
}
```

- [ ] **Step 2: Run; expect compile failure (fields don't exist yet)**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/config/...
```

- [ ] **Step 3: Add fields to `Step`**

Edit `/Users/dlo/Dev/gearup/internal/config/config.go`. Extend the `Step` struct to include the new fields. Replace the entire `Step` struct definition with:

```go
type Step struct {
	Name              string   `yaml:"name"`
	Type              string   `yaml:"type"`
	Check             string   `yaml:"check,omitempty"`
	RequiresElevation bool     `yaml:"requires_elevation,omitempty"`
	Platform          Platform `yaml:"platform,omitempty"`

	// brew-specific
	Formula string `yaml:"formula,omitempty"`

	// curl-pipe-sh
	URL   string   `yaml:"url,omitempty"`
	Shell string   `yaml:"shell,omitempty"`
	Args  []string `yaml:"args,omitempty"`

	// shell (raw)
	Install string `yaml:"install,omitempty"`
}
```

- [ ] **Step 4: Run; expect all tests pass**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/config/... -v
```

Expected: 3 tests pass (`TestProfileUnmarshal`, `TestRecipeUnmarshal`, `TestStepUnmarshal_CurlPipeAndShell`).

- [ ] **Step 5: Run the full suite to confirm no regressions**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add url, shell, args, install fields to Step"
```

---

## Task 2: `curl-pipe-sh` installer

**Files:**
- Create: `internal/installer/curlpipe/curlpipe.go`
- Create: `internal/installer/curlpipe/curlpipe_test.go`

- [ ] **Step 1: Write the failing test**

Create `/Users/dlo/Dev/gearup/internal/installer/curlpipe/curlpipe_test.go` with exactly:

```go
package curlpipe_test

import (
	"context"
	"strings"
	"testing"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer/curlpipe"
)

func TestCurlPipe_CheckInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("command -v thing").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{
		Name:  "thing",
		Type:  "curl-pipe-sh",
		URL:   "https://example.com/install.sh",
		Check: "command -v thing",
	}
	ok, err := inst.Check(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true")
	}
}

func TestCurlPipe_CheckNotInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("command -v thing").Return(gearexec.Result{ExitCode: 1}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", URL: "https://example.com/install.sh", Check: "command -v thing"}
	ok, err := inst.Check(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("installed = true, want false")
	}
}

func TestCurlPipe_MissingCheck(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", URL: "https://example.com/install.sh"}
	_, err := inst.Check(context.Background(), step)
	if err == nil {
		t.Error("want error for missing check, got nil")
	}
}

func TestCurlPipe_MissingURL(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", Check: "command -v thing"}
	err := inst.Install(context.Background(), step)
	if err == nil {
		t.Error("want error for missing url, got nil")
	}
}

func TestCurlPipe_InstallDefaultsToBash(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("curl -fsSL https://example.com/install.sh | bash").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", URL: "https://example.com/install.sh", Check: "command -v thing"}
	if err := inst.Install(context.Background(), step); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCurlPipe_InstallCustomShellAndArgs(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("curl -fsSL https://example.com/install.sh | sh -s -- --quiet").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{
		Name:  "thing",
		Type:  "curl-pipe-sh",
		URL:   "https://example.com/install.sh",
		Shell: "sh",
		Args:  []string{"--quiet"},
		Check: "command -v thing",
	}
	if err := inst.Install(context.Background(), step); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	got := f.Calls()
	if len(got) != 1 || !strings.Contains(got[0], "| sh -s -- --quiet") {
		t.Errorf("Calls = %v", got)
	}
}

func TestCurlPipe_InstallFailure(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("curl -fsSL https://example.com/install.sh | bash").Return(gearexec.Result{ExitCode: 1, Stderr: "curl: connection refused"}, nil)

	inst := &curlpipe.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "curl-pipe-sh", URL: "https://example.com/install.sh", Check: "command -v thing"}
	if err := inst.Install(context.Background(), step); err == nil {
		t.Error("want error for failed install, got nil")
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/installer/curlpipe/...
```

- [ ] **Step 3: Write the implementation**

Create `/Users/dlo/Dev/gearup/internal/installer/curlpipe/curlpipe.go` with exactly:

```go
// Package curlpipe implements the "curl-pipe-sh" step type — run a remote
// installer script by piping `curl -fsSL <url>` into a shell.
package curlpipe

import (
	"context"
	"fmt"
	"strings"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
)

// Installer is the curl-pipe-sh step installer.
type Installer struct {
	Runner gearexec.Runner
}

// Check runs the user-supplied check command; exit 0 means installed.
// curl-pipe-sh has no auto-derived check — the step MUST declare one.
func (i *Installer) Check(ctx context.Context, step config.Step) (bool, error) {
	if step.Check == "" {
		return false, fmt.Errorf("curl-pipe-sh step %q requires an explicit check", step.Name)
	}
	res, err := i.Runner.Run(ctx, step.Check)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

// Install pipes `curl -fsSL <url>` into the configured shell. The shell
// defaults to bash. If Args is non-empty, they are passed to the shell
// via `-s --`.
func (i *Installer) Install(ctx context.Context, step config.Step) error {
	if step.URL == "" {
		return fmt.Errorf("curl-pipe-sh step %q missing url", step.Name)
	}
	shell := step.Shell
	if shell == "" {
		shell = "bash"
	}
	cmd := fmt.Sprintf("curl -fsSL %s | %s", step.URL, shell)
	if len(step.Args) > 0 {
		cmd = fmt.Sprintf("%s -s -- %s", cmd, strings.Join(step.Args, " "))
	}
	res, err := i.Runner.Run(ctx, cmd)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("curl-pipe-sh %s failed (exit %d): %s", step.Name, res.ExitCode, res.Stderr)
	}
	return nil
}
```

- [ ] **Step 4: Run; expect all tests pass**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/installer/curlpipe/... -v
```

Expected: all 7 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/installer/curlpipe/
git commit -m "feat(installer/curlpipe): implement curl-pipe-sh step type"
```

---

## Task 3: `shell` installer (escape hatch)

**Files:**
- Create: `internal/installer/shell/shell.go`
- Create: `internal/installer/shell/shell_test.go`

- [ ] **Step 1: Write the failing test**

Create `/Users/dlo/Dev/gearup/internal/installer/shell/shell_test.go` with exactly:

```go
package shell_test

import (
	"context"
	"testing"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
	"gearup/internal/installer/shell"
)

func TestShell_CheckInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("test -f /opt/thing").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing", Install: "touch /opt/thing"}
	ok, err := inst.Check(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("installed = false, want true")
	}
}

func TestShell_CheckNotInstalled(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("test -f /opt/thing").Return(gearexec.Result{ExitCode: 1}, nil)

	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing", Install: "touch /opt/thing"}
	ok, err := inst.Check(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("installed = true, want false")
	}
}

func TestShell_MissingCheck(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Install: "touch /opt/thing"}
	_, err := inst.Check(context.Background(), step)
	if err == nil {
		t.Error("want error for missing check, got nil")
	}
}

func TestShell_MissingInstall(t *testing.T) {
	f := gearexec.NewFakeRunner()
	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing"}
	err := inst.Install(context.Background(), step)
	if err == nil {
		t.Error("want error for missing install, got nil")
	}
}

func TestShell_InstallSuccess(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("touch /opt/thing").Return(gearexec.Result{ExitCode: 0}, nil)

	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing", Install: "touch /opt/thing"}
	if err := inst.Install(context.Background(), step); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestShell_InstallFailure(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("touch /opt/thing").Return(gearexec.Result{ExitCode: 1, Stderr: "permission denied"}, nil)

	inst := &shell.Installer{Runner: f}
	step := config.Step{Name: "thing", Type: "shell", Check: "test -f /opt/thing", Install: "touch /opt/thing"}
	if err := inst.Install(context.Background(), step); err == nil {
		t.Error("want error for failed install, got nil")
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/installer/shell/...
```

- [ ] **Step 3: Write the implementation**

Create `/Users/dlo/Dev/gearup/internal/installer/shell/shell.go` with exactly:

```go
// Package shell implements the "shell" step type — an escape hatch for
// installers that do not map to any typed step.
package shell

import (
	"context"
	"fmt"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
)

// Installer is the shell step installer.
type Installer struct {
	Runner gearexec.Runner
}

// Check runs the user-supplied check command; exit 0 means installed.
func (i *Installer) Check(ctx context.Context, step config.Step) (bool, error) {
	if step.Check == "" {
		return false, fmt.Errorf("shell step %q requires an explicit check", step.Name)
	}
	res, err := i.Runner.Run(ctx, step.Check)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

// Install runs the user-supplied install command. Non-zero exit is an error.
func (i *Installer) Install(ctx context.Context, step config.Step) error {
	if step.Install == "" {
		return fmt.Errorf("shell step %q missing install command", step.Name)
	}
	res, err := i.Runner.Run(ctx, step.Install)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("shell step %s failed (exit %d): %s", step.Name, res.ExitCode, res.Stderr)
	}
	return nil
}
```

- [ ] **Step 4: Run; expect all tests pass**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/installer/shell/... -v
```

Expected: all 6 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/installer/shell/
git commit -m "feat(installer/shell): implement shell escape-hatch step type"
```

---

## Task 4: Register new installers in the CLI

**Files:**
- Modify: `cmd/gearup/main.go`

- [ ] **Step 1: Add imports**

Edit `/Users/dlo/Dev/gearup/cmd/gearup/main.go`. In the import block, after the existing `"gearup/internal/installer/brew"` line, add:

```go
	"gearup/internal/installer/curlpipe"
	"gearup/internal/installer/shell"
```

- [ ] **Step 2: Extend the Registry**

In `runCmd()`, replace the existing registry block:

```go
			reg := installer.Registry{
				"brew": &brew.Installer{Runner: shell},
			}
```

with:

```go
			reg := installer.Registry{
				"brew":          &brew.Installer{Runner: shellRunner},
				"curl-pipe-sh":  &curlpipe.Installer{Runner: shellRunner},
				"shell":         &shell.Installer{Runner: shellRunner},
			}
```

Note: because the new `shell` package is imported as `shell`, the existing local variable `shell := &gearexec.ShellRunner{...}` shadows the package. Rename the local variable to `shellRunner`. Update the line above it accordingly:

```go
			shellRunner := &gearexec.ShellRunner{Stdout: os.Stdout, Stderr: os.Stderr}
```

- [ ] **Step 3: Build and smoke-test**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
./gearup version
```

Expected: builds, `gearup 0.0.1-phase1` printed (version string not bumped; that will come at end of phase).

- [ ] **Step 4: Run the full test suite**

```bash
go test ./...
```

Expected: all packages pass, no regressions.

- [ ] **Step 5: Commit**

```bash
git add cmd/gearup/main.go
git commit -m "feat(cli): register curl-pipe-sh and shell installers"
```

---

## Task 5: Developer recipe + profile with the 11 tools

**Files:**
- Create: `examples/recipes/dev-stack.yaml`
- Create: `examples/profiles/dev-stack.yaml`
- Modify: `internal/profile/profile_test.go`

- [ ] **Step 1: Create the recipe**

Create `/Users/dlo/Dev/gearup/examples/recipes/dev-stack.yaml` with exactly:

```yaml
version: 1
name: dev-stack
description: "Full macOS developer toolchain: Homebrew, core CLI tools, JVM, containers, AWS, K8s, Node"

steps:
  - name: Homebrew
    type: curl-pipe-sh
    url: https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh
    check: command -v brew

  - name: Git
    type: brew
    formula: git

  - name: OpenJDK 21
    type: brew
    formula: openjdk@21

  - name: Docker CLI
    type: brew
    formula: docker

  - name: Docker Compose
    type: brew
    formula: docker-compose

  - name: Colima
    type: brew
    formula: colima

  - name: AWS CLI
    type: brew
    formula: awscli

  - name: aws-iam-authenticator
    type: brew
    formula: aws-iam-authenticator

  - name: kubectl
    type: brew
    formula: kubectl

  - name: jq
    type: brew
    formula: jq

  - name: nvm
    type: curl-pipe-sh
    url: https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.4/install.sh
    check: '[ -s "$HOME/.nvm/nvm.sh" ]'
```

- [ ] **Step 2: Create the profile**

Create `/Users/dlo/Dev/gearup/examples/profiles/dev-stack.yaml` with exactly:

```yaml
version: 1
name: "Developer Stack"
description: "Full macOS developer toolchain for a standard backend/infra stack"

platform:
  os: [darwin]

recipe_sources:
  - path: ../recipes

includes:
  - recipe: dev-stack
```

- [ ] **Step 3: Append a smoke test for the new fixture**

Append to `/Users/dlo/Dev/gearup/internal/profile/profile_test.go`:

```go
func TestResolve_DevStackFixture(t *testing.T) {
	p, err := profile.LoadProfile("../../examples/profiles/dev-stack.yaml")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	plan, err := profile.Resolve(p, "../../examples/profiles")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got, want := len(plan.Steps), 11; got != want {
		t.Fatalf("Steps len = %d, want %d", got, want)
	}
	// spot-check: first step is Homebrew (curl-pipe-sh), last step is nvm (curl-pipe-sh)
	if plan.Steps[0].Type != "curl-pipe-sh" || plan.Steps[0].Name != "Homebrew" {
		t.Errorf("Steps[0] = %+v, want Homebrew curl-pipe-sh", plan.Steps[0])
	}
	if plan.Steps[10].Type != "curl-pipe-sh" || plan.Steps[10].Name != "nvm" {
		t.Errorf("Steps[10] = %+v, want nvm curl-pipe-sh", plan.Steps[10])
	}
	// middle: OpenJDK 21 via brew
	if plan.Steps[2].Type != "brew" || plan.Steps[2].Formula != "openjdk@21" {
		t.Errorf("Steps[2] = %+v, want openjdk@21 brew", plan.Steps[2])
	}
}
```

- [ ] **Step 4: Run; expect pass**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/profile/... -v
```

Expected: 6 tests pass (the 5 from Phase 1 + the new `TestResolve_DevStackFixture`).

- [ ] **Step 5: Run the full suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add examples/recipes/dev-stack.yaml examples/profiles/dev-stack.yaml internal/profile/profile_test.go
git commit -m "feat(examples): add developer stack recipe with 11 tools"
```

---

## Task 6: Bump version and update README

**Files:**
- Modify: `cmd/gearup/main.go` (version constant)
- Modify: `README.md`

- [ ] **Step 1: Bump the version**

In `/Users/dlo/Dev/gearup/cmd/gearup/main.go`, change:

```go
const version = "0.0.1-phase1"
```

to:

```go
const version = "0.0.2-phase2"
```

- [ ] **Step 2: Update the README**

Replace the Phase 1 status line and usage section in `/Users/dlo/Dev/gearup/README.md`.

Replace:

```markdown
Status: Phase 1 (tracer bullet) — in development.
```

with:

```markdown
Status: Phase 2 — in development. Supports `brew`, `curl-pipe-sh`, and `shell` step types on macOS.
```

Replace:

```markdown
## Phase 1 usage

    gearup run --profile ./examples/profiles/example.yaml

Requires macOS with Homebrew installed. Only the `brew` step type is supported in Phase 1.
```

with:

```markdown
## Usage

    gearup run --profile ./examples/profiles/dev-stack.yaml

Requires macOS. If Homebrew is not installed, the profile's first step installs it via the official installer. Subsequent brew steps in the same run require `brew` on PATH — if Homebrew was just installed, open a new shell and re-run so PATH picks up `/opt/homebrew/bin` (or `/usr/local/bin` on Intel).
```

- [ ] **Step 3: Rebuild and verify**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
./gearup version
```

Expected: `gearup 0.0.2-phase2`.

- [ ] **Step 4: Run full suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/gearup/main.go README.md
git commit -m "chore: bump version to 0.0.2-phase2 and update README for Phase 2"
```

---

## Task 7: Manual verification (user)

The user runs the binary against the real `dev-stack` profile on their Mac.

- [ ] **Step 1: Build**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
```

- [ ] **Step 2: First run**

```bash
./gearup run --profile ./examples/profiles/dev-stack.yaml
```

Expected behavior on a machine that already has Homebrew:
- Homebrew check passes → skip.
- Each brew formula: check passes (if already installed) → skip, OR check fails → install.
- nvm check: passes if `~/.nvm/nvm.sh` exists → skip; otherwise curl-pipes the installer.

On a fully clean machine (no Homebrew):
- Homebrew step installs Homebrew (interactively — user types sudo password).
- **Subsequent brew steps will fail** because `brew` isn't on PATH in gearup's process. The user must open a new shell and re-run. This is documented in the README and is a known Phase 2 limitation.

- [ ] **Step 3: Second run — idempotency**

```bash
./gearup run --profile ./examples/profiles/dev-stack.yaml
```

Expected: every step reports `already installed — skip`, no installer invocations.

- [ ] **Step 4: Exit code**

```bash
./gearup run --profile ./examples/profiles/dev-stack.yaml; echo "exit=$?"
```

Expected: `exit=0`.

---

## Phase 2 completion criteria

- [ ] `go test ./...` passes (expected: all 6 packages; one new package `curlpipe`, one new `shell`).
- [ ] `gearup version` prints `gearup 0.0.2-phase2`.
- [ ] `gearup run --profile ./examples/profiles/dev-stack.yaml` completes successfully against the user's real Mac.
- [ ] On re-run, every step is skipped.
- [ ] No proprietary content anywhere in the repo. (Scan before tagging.)

---

## Known Phase 2 limitations (addressed in later phases)

- **Mid-run PATH for freshly-installed Homebrew.** If Homebrew is installed during a `gearup run`, subsequent brew steps fail because `brew` isn't on PATH in gearup's process. Workaround: re-run gearup after opening a new shell. Proper fix is a PATH bootstrap in a later phase (likely Phase 5 with `post_install` or a dedicated mechanism in Phase 6/7).
- **No elevation banner.** Homebrew's installer will prompt for sudo directly; there's no gearup-level banner. Phase 3 adds that.
- **curl-pipe-sh output streams live.** Same cosmetic issue as Phase 1 (check-command output). Phase 4 UX polish addresses it.

---

## Self-review

**Spec coverage:**
- ✅ `curl-pipe-sh` step type — Task 2
- ✅ `shell` step type — Task 3
- ✅ Registry integration — Task 4
- ✅ Developer recipe with 11 tools — Task 5
- ✅ Version bump + README — Task 6
- ✅ Manual verification — Task 7

**Placeholder scan:** No "TBD" / "TODO" / "similar to". Every task has full code and exact commands.

**Type consistency:** `config.Step` gains `URL`, `Shell`, `Args`, `Install` — all referenced consistently in curlpipe and shell tests. The main.go variable rename (`shell` → `shellRunner`) is explicitly called out in Task 4 Step 2 to avoid package/local-variable shadow collision.

**Trade-offs (intentional, deferred):**
- No PATH bootstrap for mid-run Homebrew installs — documented as a limitation.
- `curl-pipe-sh` does not support piping into non-bash/sh shells beyond what's declared in `step.Shell`. No tests for arbitrary shells (`zsh`, `python`, etc.) because no realistic use case in Phase 2.
- `shell` step does not support a separate working directory or env. If needed, callers embed `cd` into the `install` string.
