# gearup Phase 4A — Silent checks, log file, dry-run

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Kill the wall of `brew list` / `/opt/homebrew/Cellar/...` spam that check commands dump into the terminal, by capturing all command output to a per-run log file while only install commands stream live. Add `gearup plan` / `--dry-run` to preview what would run, and a `--yes` flag for non-interactive use.

**Architecture:** The `exec.Runner` interface gains a second method `RunQuiet` alongside `Run`. Installers call `RunQuiet` for their check phase and `Run` for their install phase. A new `internal/log` package owns per-run log files under `$XDG_STATE_HOME/gearup/logs/`. The `ShellRunner` has three output sinks (stream stdout, stream stderr, log) and multiplexes appropriately per method. The CLI opens a log file per `gearup run` and wires it through. A new `--dry-run` flag flips the runner into a check-only mode that prints a plan and exits 10 if any step would run.

**Tech Stack:** Go, cobra, yaml.v3, lipgloss, huh. Stdlib only for log-file writing and TTY detection (`os.Stdin.Stat()` + ModeCharDevice).

**Spec reference:** `docs/superpowers/specs/2026-04-15-gearup-design.md` §4.4 (dry-run), §4.7 (logging), §5 Phase 4.

---

## Scope

**In Phase 4A:**
- Two-method `exec.Runner`: `Run` (live-streamed + logged) and `RunQuiet` (logged only, not streamed).
- `ShellRunner` with `StreamOut`/`StreamErr`/`LogOut` sinks; all nil-safe.
- `FakeRunner` gains `RunQuiet`; `Calls()` exposes per-call mode.
- All three installers (brew, curlpipe, shell) use `RunQuiet` for checks and `Run` for installs.
- New `internal/log` package: opens `$XDG_STATE_HOME/gearup/logs/<timestamp>-<recipe>.log`, writes header, provides `io.Writer`.
- On step failure, the runner reports the log file path alongside the error.
- `gearup run --dry-run` and its alias `gearup plan`. Both exit 0 if no step would run, 10 if any step would run (CI-friendly).
- `--yes` flag: swaps the elevation prompter for `AutoApprovePrompter` (silently continues).
- TTY guard at the top of `run` (non-dry-run): error out cleanly if stdin is not a terminal unless `--yes` is set.
- Version bump `0.0.5-phase4a`; README updated.

**Out of Phase 4A (deferred to Phase 4B):**
- Lip Gloss plan-preview table.
- Huh recipe picker.
- Bubbles spinner.
- Elevation-banner suppression when no elevation step would actually fire.

---

## Design notes

### Runner interface change

```go
type Runner interface {
    Run(ctx context.Context, cmd string) (Result, error)       // streamed + logged
    RunQuiet(ctx context.Context, cmd string) (Result, error)  // logged only
}
```

Installer usage:
- `Check(...)` → `RunQuiet(...)`
- `Install(...)` → `Run(...)`

### ShellRunner sinks

```go
type ShellRunner struct {
    StreamOut io.Writer   // e.g., os.Stdout. Nil = not streamed.
    StreamErr io.Writer   // e.g., os.Stderr. Nil = not streamed.
    LogOut    io.Writer   // per-run log file. Nil = no log.
}
```

- `Run` fans stdout to `MultiWriter(StreamOut, LogOut, captureBuf)` with any nil skipped; stderr to `MultiWriter(StreamErr, LogOut, captureBuf)`.
- `RunQuiet` fans stdout/stderr to `MultiWriter(LogOut, captureBuf)` — never to `StreamOut`/`StreamErr`.

### Dry-run plumbing

- `Runner.DryRun bool` flag.
- When true, `Runner.Run(ctx, plan)` skips elevation acquisition entirely and only calls each installer's `Check`. Prints one line per step: `already installed` or `WOULD install`. Collects a count.
- Returns a sentinel error `runner.ErrDryRunPending` when at least one step would run. `main.go` checks `errors.Is(err, runner.ErrDryRunPending)` and exits 10.

### TTY guard

- At the top of `runCmd`'s RunE, before loading anything, check `os.Stdin.Stat().Mode() & os.ModeCharDevice`.
- If not a TTY AND not `--dry-run` AND not `--yes` → print guidance to stderr and exit 3.
- `--dry-run` path never needs TTY (no Huh prompts).
- `--yes` with a non-TTY is fine (scripted use).

### `--yes` behavior

- Add `AutoApprovePrompter` to the `internal/elevation` package. Its `Confirm` always returns `(true, nil)`.
- In the CLI, if `--yes` is set, wire `elevation.AutoApprovePrompter{}` instead of `HuhPrompter{}`.
- The Lip Gloss banner still prints (cheap, single write) so scripted runs still have an audit trail of what was acknowledged.

---

## File structure (changes)

```
gearup/
├── internal/
│   ├── exec/
│   │   ├── exec.go                     # MODIFY: add RunQuiet, restructure fields
│   │   ├── exec_test.go                # MODIFY: RunQuiet tests, update existing tests
│   │   └── fake.go                     # MODIFY: implement RunQuiet; Calls returns []FakeCall
│   ├── log/
│   │   ├── log.go                      # NEW
│   │   └── log_test.go                 # NEW
│   ├── elevation/
│   │   └── auto.go                     # NEW: AutoApprovePrompter
│   ├── installer/
│   │   ├── brew/brew.go                # MODIFY: Check → RunQuiet
│   │   ├── brew/brew_test.go           # MODIFY: updated FakeRunner Calls shape
│   │   ├── curlpipe/curlpipe.go        # MODIFY: Check → RunQuiet
│   │   ├── curlpipe/curlpipe_test.go   # MODIFY
│   │   ├── shell/shell.go              # MODIFY: Check → RunQuiet
│   │   └── shell/shell_test.go         # MODIFY
│   └── runner/
│       ├── runner.go                   # MODIFY: DryRun path, ErrDryRunPending, log-path error enrichment
│       └── runner_test.go              # MODIFY: add dry-run tests; update FakeRunner Calls references
├── cmd/gearup/
│   └── main.go                         # MODIFY: --dry-run, plan subcommand, --yes, TTY guard, log wiring, bump version
├── README.md                           # MODIFY
└── plans/
    └── 2026-04-15-phase-4a-silent-checks-and-dry-run.md   # this file
```

---

## Task 1: Extend the exec Runner interface with `RunQuiet`

**Files:**
- Modify: `internal/exec/exec.go`
- Modify: `internal/exec/fake.go`
- Modify: `internal/exec/exec_test.go`

- [ ] **Step 1: Write failing tests**

Replace the contents of `/Users/dlo/Dev/gearup/internal/exec/exec_test.go` with:

```go
package exec_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	gearexec "gearup/internal/exec"
)

func TestShellRunner_Run_StreamsAndLogs(t *testing.T) {
	var stream, log bytes.Buffer
	r := &gearexec.ShellRunner{StreamOut: &stream, LogOut: &log}
	res, err := r.Run(context.Background(), "echo hello-gearup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if !strings.Contains(stream.String(), "hello-gearup") {
		t.Errorf("stream missing output: %q", stream.String())
	}
	if !strings.Contains(log.String(), "hello-gearup") {
		t.Errorf("log missing output: %q", log.String())
	}
}

func TestShellRunner_RunQuiet_LogsButDoesNotStream(t *testing.T) {
	var stream, log bytes.Buffer
	r := &gearexec.ShellRunner{StreamOut: &stream, LogOut: &log}
	res, err := r.RunQuiet(context.Background(), "echo should-be-quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if stream.Len() != 0 {
		t.Errorf("stream should be empty, got: %q", stream.String())
	}
	if !strings.Contains(log.String(), "should-be-quiet") {
		t.Errorf("log missing output: %q", log.String())
	}
}

func TestShellRunner_Run_NilSinksAreSafe(t *testing.T) {
	r := &gearexec.ShellRunner{}
	res, err := r.Run(context.Background(), "echo ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Stdout, "ok") {
		t.Errorf("captured stdout should still contain echo output: %q", res.Stdout)
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

func TestFakeRunner_Run_ReturnsProgrammedResult(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 0, Stdout: "/opt/homebrew/Cellar/jq"}, nil)
	res, err := f.Run(context.Background(), "brew list --formula jq")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if got := f.Calls(); len(got) != 1 || got[0].Cmd != "brew list --formula jq" || got[0].Quiet {
		t.Errorf("Calls = %+v, want one non-quiet call to brew list", got)
	}
}

func TestFakeRunner_RunQuiet_MarksCallAsQuiet(t *testing.T) {
	f := gearexec.NewFakeRunner()
	f.On("brew list --formula jq").Return(gearexec.Result{ExitCode: 0}, nil)
	_, err := f.RunQuiet(context.Background(), "brew list --formula jq")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := f.Calls()
	if len(got) != 1 {
		t.Fatalf("Calls len = %d, want 1", len(got))
	}
	if !got[0].Quiet {
		t.Errorf("Calls[0].Quiet = false, want true")
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

- [ ] **Step 2: Run tests; expect compile failure**

```bash
cd /Users/dlo/Dev/gearup
go test ./internal/exec/...
```

- [ ] **Step 3: Replace `internal/exec/exec.go`**

Replace the entire contents of `/Users/dlo/Dev/gearup/internal/exec/exec.go` with:

```go
// Package exec wraps os/exec with a Runner interface that installers depend
// on. The interface exposes two modes:
//
//   - Run: streams output live (to StreamOut/StreamErr, typically the terminal)
//     AND mirrors it to LogOut (typically the per-run log file).
//   - RunQuiet: mirrors output to LogOut only; does NOT stream live. Used for
//     check commands whose output is noise under normal operation.
//
// Both methods also capture output into the returned Result for callers that
// need to inspect stdout/stderr programmatically.
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

// Runner runs shell commands. Non-zero exit is reported in Result.ExitCode,
// NOT as an error; err is non-nil only for spawn failures.
type Runner interface {
	// Run streams stdout/stderr live (when StreamOut/StreamErr are set) AND
	// mirrors them to LogOut. Used for install commands where the user
	// wants to watch progress.
	Run(ctx context.Context, cmd string) (Result, error)

	// RunQuiet mirrors stdout/stderr to LogOut only. Used for check commands
	// whose output is noise in the terminal but useful in a log file.
	RunQuiet(ctx context.Context, cmd string) (Result, error)
}

// ShellRunner runs commands via `bash -c` with three independent output sinks.
// Any nil sink is skipped. Regardless of sink configuration, stdout and stderr
// are always captured into the returned Result.
type ShellRunner struct {
	StreamOut io.Writer // e.g., os.Stdout
	StreamErr io.Writer // e.g., os.Stderr
	LogOut    io.Writer // per-run log file; receives every byte of both modes
}

// Run implements Runner with live streaming.
func (r *ShellRunner) Run(ctx context.Context, cmd string) (Result, error) {
	return r.runWith(ctx, cmd, true)
}

// RunQuiet implements Runner without live streaming.
func (r *ShellRunner) RunQuiet(ctx context.Context, cmd string) (Result, error) {
	return r.runWith(ctx, cmd, false)
}

func (r *ShellRunner) runWith(ctx context.Context, cmd string, stream bool) (Result, error) {
	c := osexec.CommandContext(ctx, "bash", "-c", cmd)
	var outBuf, errBuf bytes.Buffer
	c.Stdout = fanout(stream, r.StreamOut, r.LogOut, &outBuf)
	c.Stderr = fanout(stream, r.StreamErr, r.LogOut, &errBuf)
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

// fanout builds an io.Writer that writes to all non-nil sinks.
// If stream is false, streamSink is excluded even if non-nil.
// The capture buffer is always included.
func fanout(stream bool, streamSink, logSink io.Writer, capture *bytes.Buffer) io.Writer {
	var ws []io.Writer
	if stream && streamSink != nil {
		ws = append(ws, streamSink)
	}
	if logSink != nil {
		ws = append(ws, logSink)
	}
	ws = append(ws, capture)
	return io.MultiWriter(ws...)
}
```

- [ ] **Step 4: Replace `internal/exec/fake.go`**

Replace the entire contents of `/Users/dlo/Dev/gearup/internal/exec/fake.go` with:

```go
package exec

import (
	"context"
	"fmt"
	"sync"
)

// FakeCall records a command invocation on FakeRunner, including which
// method (Run vs RunQuiet) was used.
type FakeCall struct {
	Cmd   string
	Quiet bool
}

// FakeRunner is a test double for Runner. Register expected commands with
// On().Return(...); any call to an unstubbed command returns an error.
type FakeRunner struct {
	mu    sync.Mutex
	stubs map[string]fakeResponse
	calls []FakeCall
}

type fakeResponse struct {
	result Result
	err    error
}

// NewFakeRunner returns an empty FakeRunner.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{stubs: map[string]fakeResponse{}}
}

type stubBuilder struct {
	f   *FakeRunner
	cmd string
}

// On starts a stub registration for the given command string.
func (f *FakeRunner) On(cmd string) *stubBuilder { return &stubBuilder{f: f, cmd: cmd} }

// Return sets the Result and error returned when the stubbed command runs.
func (b *stubBuilder) Return(res Result, err error) {
	b.f.mu.Lock()
	defer b.f.mu.Unlock()
	b.f.stubs[b.cmd] = fakeResponse{result: res, err: err}
}

// Run implements Runner (streamed).
func (f *FakeRunner) Run(_ context.Context, cmd string) (Result, error) {
	return f.dispatch(cmd, false)
}

// RunQuiet implements Runner (quiet).
func (f *FakeRunner) RunQuiet(_ context.Context, cmd string) (Result, error) {
	return f.dispatch(cmd, true)
}

func (f *FakeRunner) dispatch(cmd string, quiet bool) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, FakeCall{Cmd: cmd, Quiet: quiet})
	resp, ok := f.stubs[cmd]
	if !ok {
		return Result{}, fmt.Errorf("FakeRunner: unstubbed command %q", cmd)
	}
	return resp.result, resp.err
}

// Calls returns the commands that have been invoked, in order, with their modes.
func (f *FakeRunner) Calls() []FakeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]FakeCall(nil), f.calls...)
}
```

- [ ] **Step 5: Run tests; expect PASS on exec package**

```bash
go test ./internal/exec/... -v
```

Expected: 7 tests pass (3 ShellRunner Run/Quiet/nil-safe, 1 false-exit, 2 FakeRunner Run/Quiet, 1 unstubbed).

- [ ] **Step 6: Other packages will fail** because installers still use the old `Run` semantics for checks, and their tests call `Calls()` returning strings.

```bash
go test ./...
```

Expected: broken in `installer/brew`, `installer/curlpipe`, `installer/shell`, `runner`. Task 2 fixes all of them.

- [ ] **Step 7: Commit**

```bash
git add internal/exec/
git commit -m "feat(exec): add RunQuiet; split stream vs log sinks on ShellRunner"
```

---

## Task 2: Update installers to use `RunQuiet` for checks

**Files:**
- Modify: `internal/installer/brew/brew.go` and `brew_test.go`
- Modify: `internal/installer/curlpipe/curlpipe.go` and `curlpipe_test.go`
- Modify: `internal/installer/shell/shell.go` and `shell_test.go`

- [ ] **Step 1: brew installer — production code**

In `/Users/dlo/Dev/gearup/internal/installer/brew/brew.go`, change the one `i.Runner.Run` call inside `Check` to `i.Runner.RunQuiet`. The `Install` method's `i.Runner.Run` stays as-is.

- [ ] **Step 2: brew installer — tests**

`internal/installer/brew/brew_test.go` uses `f.Calls()` which now returns `[]FakeCall`. No existing test in this file asserts on `f.Calls()` directly (they only assert on return values and stubs), so the test file should compile as-is. Run:

```bash
go test ./internal/installer/brew/... -v
```

If any test fails, inspect the compilation/assertion error. If `Calls()` is referenced, update `got[0]` → `got[0].Cmd`. If a test cares about quiet-mode, add a `got[0].Quiet` assertion.

Also — `TestBrew_CheckUsesOverrideWhenSet` asserts the override path is taken. Update its stub to match: the check runs via `RunQuiet`, so the stub command is the same ("command -v git"), but the assertion on `f.Calls()` needs updating:

Find:
```go
if got := f.Calls(); len(got) != 1 || got[0] != "command -v git" {
    t.Errorf("Calls = %v, want single call to 'command -v git' (not default)", got)
}
```

Replace with:
```go
if got := f.Calls(); len(got) != 1 || got[0].Cmd != "command -v git" || !got[0].Quiet {
    t.Errorf("Calls = %+v, want single quiet call to 'command -v git'", got)
}
```

Run again — should pass.

- [ ] **Step 3: curlpipe installer — production + tests**

In `/Users/dlo/Dev/gearup/internal/installer/curlpipe/curlpipe.go`, change the `i.Runner.Run` call inside `Check` to `i.Runner.RunQuiet`. `Install` stays on `Run`.

In `curlpipe_test.go`, find the assertion:
```go
got := f.Calls()
if len(got) != 1 || !strings.Contains(got[0], "| sh -s -- --quiet") {
    t.Errorf("Calls = %v", got)
}
```

Replace with:
```go
got := f.Calls()
if len(got) != 1 || !strings.Contains(got[0].Cmd, "| sh -s -- --quiet") {
    t.Errorf("Calls = %+v", got)
}
```

Run:
```bash
go test ./internal/installer/curlpipe/... -v
```

Expected: 7 tests pass.

- [ ] **Step 4: shell installer — production + tests**

Same edit pattern in `/Users/dlo/Dev/gearup/internal/installer/shell/shell.go`: `Check` calls `RunQuiet`, `Install` calls `Run`.

`shell_test.go` uses `Calls()`? Scan the file. If any `got[0]` access exists, update to `got[0].Cmd`. If no such access, no changes.

Run:
```bash
go test ./internal/installer/shell/... -v
```

Expected: 6 tests pass.

- [ ] **Step 5: Full suite check**

```bash
go test ./...
```

Expected: runner package will still be broken until Task 3 wiring. That's OK for this task's commit — we're committing installer-level changes.

Actually — the runner package depends on installer which depends on exec. Since we haven't changed runner yet and installers only moved from Run to RunQuiet, runner tests that use `recordingInstaller` or `installerFunc` still work (those fakes don't call the real Runner). The existing runner-level FakeRunner usage needs its `Calls()` assertion updated.

Check: does `runner_test.go` call `Calls()` anywhere? If yes, update `got[0]` → `got[0].Cmd`. If no, runner tests should still pass.

Run `go test ./internal/runner/...` — report failures to fix in this task if they're just signature mismatches; defer structural runner changes to Task 3.

- [ ] **Step 6: Commit**

```bash
git add internal/installer/
git commit -m "refactor(installers): use RunQuiet for check commands"
```

If runner tests were updated too:

```bash
git add internal/runner/
git commit -m "test(runner): update FakeCall assertions"
```

(Or combine; your call.)

---

## Task 3: `internal/log` package

**Files:**
- Create: `internal/log/log.go`
- Create: `internal/log/log_test.go`

- [ ] **Step 1: Failing test**

Create `/Users/dlo/Dev/gearup/internal/log/log_test.go`:

```go
package log_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gearlog "gearup/internal/log"
)

func TestCreate_OpensFileWritesHeader(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	lf, err := gearlog.Create("backend")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer lf.Close()

	if lf.Path() == "" {
		t.Fatal("Path() is empty")
	}
	if !strings.Contains(lf.Path(), filepath.Join("gearup", "logs")) {
		t.Errorf("Path() = %q, want to contain gearup/logs", lf.Path())
	}
	if !strings.Contains(lf.Path(), "backend") {
		t.Errorf("Path() = %q, want to contain recipe name", lf.Path())
	}

	if _, err := lf.Writer().Write([]byte("hello log\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := lf.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	data, err := os.ReadFile(lf.Path())
	if err != nil {
		t.Fatalf("readfile: %v", err)
	}
	if !strings.Contains(string(data), "recipe: backend") {
		t.Errorf("log missing header, got: %q", string(data))
	}
	if !strings.Contains(string(data), "hello log") {
		t.Errorf("log missing written content, got: %q", string(data))
	}
}

func TestCreate_FallsBackToHomeWhenXDGUnset(t *testing.T) {
	// Simulate a user without XDG_STATE_HOME. HOME is set to a temp dir so we
	// don't touch the real filesystem.
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", dir)

	lf, err := gearlog.Create("frontend")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer lf.Close()

	want := filepath.Join(dir, ".local", "state", "gearup", "logs")
	if !strings.HasPrefix(lf.Path(), want) {
		t.Errorf("Path() = %q, want prefix %q", lf.Path(), want)
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

```bash
go test ./internal/log/...
```

- [ ] **Step 3: Implementation**

Create `/Users/dlo/Dev/gearup/internal/log/log.go`:

```go
// Package log owns per-run log files. Each gearup run opens a timestamped
// file under $XDG_STATE_HOME/gearup/logs/ (falling back to
// ~/.local/state/gearup/logs/) and writes a short header. Installers and
// the runner mirror command output into this file for audit + debugging.
package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File wraps an open log file handle.
type File struct {
	f    *os.File
	path string
}

// Create opens a new per-run log file named after recipe + current timestamp,
// writes a header identifying the recipe and run time, and returns the handle.
func Create(recipeName string) (*File, error) {
	dir, err := logDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	safe := sanitize(recipeName)
	name := fmt.Sprintf("%s-%s.log", time.Now().Format("20060102-150405"), safe)
	path := filepath.Join(dir, name)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	fmt.Fprintf(f, "# gearup run log\n# recipe: %s\n# started: %s\n\n", recipeName, time.Now().Format(time.RFC3339))
	return &File{f: f, path: path}, nil
}

// Writer returns the underlying io.Writer for mirroring command output.
func (l *File) Writer() io.Writer { return l.f }

// Path returns the absolute path of the log file.
func (l *File) Path() string { return l.path }

// Close flushes and closes the log file.
func (l *File) Close() error { return l.f.Close() }

func logDir() (string, error) {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "gearup", "logs"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "gearup", "logs"), nil
}

// sanitize replaces filesystem-unfriendly characters in the recipe name so it
// can appear in a filename. It is intentionally conservative.
func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unnamed"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
```

- [ ] **Step 4: Run; expect PASS**

```bash
go test ./internal/log/... -v
```

Expected: 2 tests pass.

- [ ] **Step 5: Full suite**

```bash
go test ./...
```

Expected: all packages pass (assuming Task 2 left them passing).

- [ ] **Step 6: Commit**

```bash
git add internal/log/
git commit -m "feat(log): add per-run log file under XDG_STATE_HOME/gearup/logs"
```

---

## Task 4: Add `AutoApprovePrompter` to elevation

**Files:**
- Create: `internal/elevation/auto.go`
- Modify: `internal/elevation/elevation_test.go`

- [ ] **Step 1: Failing test**

Append to `/Users/dlo/Dev/gearup/internal/elevation/elevation_test.go`:

```go
func TestAutoApprovePrompter_AlwaysConfirms(t *testing.T) {
	p := elevation.AutoApprovePrompter{}
	ok, err := p.Confirm("whatever")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("Confirm returned false, want true")
	}
}
```

- [ ] **Step 2: Run; expect compile failure**

- [ ] **Step 3: Implementation**

Create `/Users/dlo/Dev/gearup/internal/elevation/auto.go`:

```go
package elevation

// AutoApprovePrompter always returns (true, nil). Used for --yes / scripted
// runs where interactive confirmation is undesirable. The banner is still
// printed upstream, so the run log records what was auto-approved.
type AutoApprovePrompter struct{}

// Confirm implements Prompter.
func (AutoApprovePrompter) Confirm(_ string) (bool, error) { return true, nil }
```

- [ ] **Step 4: Run; expect PASS**

```bash
go test ./internal/elevation/... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/elevation/
git commit -m "feat(elevation): add AutoApprovePrompter for --yes"
```

---

## Task 5: Runner dry-run mode

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/runner_test.go`

- [ ] **Step 1: Failing tests**

Append to `/Users/dlo/Dev/gearup/internal/runner/runner_test.go`:

```go
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
```

Add `"errors"` to the import block of `runner_test.go` if not already present.

- [ ] **Step 2: Run; expect compile failure**

- [ ] **Step 3: Update `internal/runner/runner.go`**

Replace the entire contents of `/Users/dlo/Dev/gearup/internal/runner/runner.go` with:

```go
// Package runner executes a ResolvedPlan by dispatching each step to its
// registered installer. It stops on the first error (fail-fast). When the
// plan's Recipe declares an Elevation block and at least one step requires
// elevation, the runner batches elevation-required steps after acquiring
// confirmation from the user. In dry-run mode the runner only calls each
// step's Check and prints a plan, returning ErrDryRunPending if any step
// would be installed.
package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gearup/internal/config"
	"gearup/internal/elevation"
	"gearup/internal/installer"
)

// ErrDryRunPending is returned by Run when DryRun is true and at least one
// step reports it would be installed. The CLI uses this to exit with code 10,
// a CI-friendly signal that the machine is not fully provisioned.
var ErrDryRunPending = errors.New("dry-run: one or more steps would install")

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
	DryRun   bool               // when true, only checks run; no installs
}

const expiryWarnThreshold = 30 * time.Second

// Run walks plan.Steps. See package doc for the modes.
func (r *Runner) Run(ctx context.Context, plan *config.ResolvedPlan) error {
	if r.DryRun {
		return r.runDryRun(ctx, plan)
	}
	return r.runLive(ctx, plan)
}

func (r *Runner) runDryRun(ctx context.Context, plan *config.ResolvedPlan) error {
	total := len(plan.Steps)
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
		if installed {
			r.Out.Printf("[%d/%d] %s: already installed\n", idx, total, step.Name)
		} else {
			r.Out.Printf("[%d/%d] %s: WOULD install\n", idx, total, step.Name)
			willInstall++
		}
	}
	r.Out.Printf("\nSummary: %d to install · %d already installed\n", willInstall, total-willInstall)
	if willInstall > 0 {
		return ErrDryRunPending
	}
	return nil
}

func (r *Runner) runLive(ctx context.Context, plan *config.ResolvedPlan) error {
	elevSteps, regSteps := partition(plan)
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

- [ ] **Step 4: Run; expect PASS**

```bash
go test ./internal/runner/... -v
```

Expected: all prior tests + 3 new dry-run tests pass.

- [ ] **Step 5: Full suite**

```bash
go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/runner/
git commit -m "feat(runner): add DryRun mode; ErrDryRunPending sentinel"
```

---

## Task 6: CLI wiring — `--dry-run`, `plan`, `--yes`, TTY guard, log file, version bump

**Files:**
- Modify: `cmd/gearup/main.go`

- [ ] **Step 1: Rewrite `main.go`**

Replace the entire contents of `/Users/dlo/Dev/gearup/cmd/gearup/main.go` with:

```go
// Command gearup is an open-source macOS developer-machine bootstrap CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
)

const version = "0.0.5-phase4a"

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
	cmd.Flags().StringVar(&recipePath, "recipe", "", "path to recipe YAML (required)")
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
	cmd.Flags().StringVar(&recipePath, "recipe", "", "path to recipe YAML (required)")
	return cmd
}

func execute(recipePath string, dryRun, yes bool) error {
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "gearup currently supports macOS only")
		os.Exit(4)
	}
	if recipePath == "" {
		return fmt.Errorf("--recipe is required")
	}

	// TTY guard: interactive runs (non-dry-run, non-yes) require a terminal
	// because the elevation confirm blocks on Huh.
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

	// Open a per-run log file for command output mirroring.
	lf, err := gearlog.Create(rec.Name)
	if err != nil {
		return err
	}
	defer lf.Close()

	// Build the shared ShellRunner: stream to terminal, mirror to log.
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

	r := &runner.Runner{
		Registry: reg,
		Out:      stdPrinter{},
		Prompter: prompter,
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
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr, "full log:", lf.Path())
		os.Exit(1)
	}
	if !dryRun {
		fmt.Println("\nDone.")
	}
	return nil
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

// isTerminal reports whether f is a character device (interactive TTY).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// stdPrinter adapts os.Stdout to runner.Writer.
type stdPrinter struct{}

func (stdPrinter) Printf(format string, args ...any) { fmt.Printf(format, args...) }
func (stdPrinter) Write(p []byte) (int, error)       { return os.Stdout.Write(p) }

// ensure io.Writer import is used
var _ io.Writer = stdPrinter{}
```

Note the `installer/shell` import alias `installshell` to avoid collision with a (hypothetical) `shell` local — and the local `shellRunner` var name. Adjust the `shell` package import alias as shown (`installshell "gearup/internal/installer/shell"`).

- [ ] **Step 2: Build + spot-checks**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup

./gearup version
# Expected: gearup 0.0.5-phase4a

./gearup run
# Expected: "Error: --recipe is required", exit 1

./gearup plan --recipe ./examples/recipes/frontend.yaml
# Expected: PLAN (dry-run) header, each step as "already installed" (since you have them), summary, exit 0.

./gearup plan --recipe ./examples/recipes/frontend.yaml; echo "exit=$?"
# Expected exit=0 because nothing would run.
```

- [ ] **Step 3: Full test suite**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/gearup/main.go
git commit -m "feat(cli): add --dry-run, plan subcommand, --yes, TTY guard, per-run log"
```

---

## Task 7: README update

**File:** `README.md`

- [ ] **Step 1: Update status line**

Find:
```
Status: Phase 3 — in development. Supports three step types, per-step `requires_elevation` with a configurable banner + confirmation, on macOS.
```

Replace with:
```
Status: Phase 4A — in development. Silent check commands (logged, not streamed), `--dry-run` / `gearup plan`, `--yes` for scripted use, per-run log file at `$XDG_STATE_HOME/gearup/logs/`.
```

- [ ] **Step 2: Add a new subsection under Usage**

After the existing `### Elevation` subsection, insert:

```markdown

### Preview a recipe without running it

    gearup plan --recipe ./examples/recipes/backend.yaml

Runs every step's `check` and prints what would happen, without installing anything. Exits 0 if the machine is already provisioned, or 10 if any step would run (CI-friendly: assert your machine matches the recipe).

### Scripted / non-interactive use

    gearup run --recipe ./examples/recipes/backend.yaml --yes

`--yes` auto-approves the elevation confirmation so CI / scripts don't block. Combine with `--dry-run` for non-destructive CI checks.

### Log files

Every `gearup run` writes a log file at `$XDG_STATE_HOME/gearup/logs/<timestamp>-<recipe>.log` (falling back to `~/.local/state/gearup/logs/`). Check-command output is logged but not shown on the terminal; install-command output is both streamed live and mirrored to the log. On step failure, the log path is printed to stderr.
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document Phase 4A features (dry-run, --yes, log file)"
```

---

## Task 8: Manual verification (user)

The user runs the recipes on their Mac and confirms the new output is clean.

- [ ] **Step 1: Build**

```bash
cd /Users/dlo/Dev/gearup
go build -o gearup ./cmd/gearup
./gearup version
```

Expected: `gearup 0.0.5-phase4a`.

- [ ] **Step 2: Silent checks in action**

```bash
./gearup run --recipe ./examples/recipes/backend.yaml
```

Expected output is dramatically cleaner than Phase 3: for each step, just three lines max:

```
[N/12] Step Name: checking...
[N/12] Step Name: already installed — skip
```

No walls of `/opt/homebrew/Cellar/...` — that output now lives only in the log file.

Find the log:

```bash
ls ~/.local/state/gearup/logs/ | tail -1
# Or:
tail -50 ~/.local/state/gearup/logs/$(ls ~/.local/state/gearup/logs/ | tail -1)
```

Expected: the brew-list output from every check is captured in the log.

- [ ] **Step 3: Dry-run**

```bash
./gearup plan --recipe ./examples/recipes/backend.yaml; echo "exit=$?"
```

Expected: `PLAN (dry-run) · Backend (12 steps)` header, each step prefixed with `[N/12]` reporting `already installed` or `WOULD install`, summary footer, exit 0 (assuming your machine is fully provisioned).

To see the non-zero exit path, pick a small tool you don't have — e.g., temporarily remove one via `brew uninstall jq` and re-run `plan`. Expect:

```
[10/12] jq: WOULD install
Summary: 1 to install · 11 already installed
note: re-run without --dry-run to apply the pending installs
exit=10
```

Then `brew install jq` to restore (or let the next `gearup run` handle it).

- [ ] **Step 4: `--yes` bypasses elevation prompt**

```bash
./gearup run --recipe ./examples/recipes/backend.yaml --yes
```

Expected: banner still prints, but no Huh confirm pauses for input. The JVM symlink step is auto-approved (skip since it already exists). Run completes, exit 0.

- [ ] **Step 5: TTY guard**

```bash
./gearup run --recipe ./examples/recipes/backend.yaml < /dev/null; echo "exit=$?"
```

Expected: error message about requiring a terminal, exit 3.

And:

```bash
./gearup plan --recipe ./examples/recipes/backend.yaml < /dev/null; echo "exit=$?"
```

Expected: works fine (dry-run bypasses TTY guard). Exit 0.

- [ ] **Step 6: Inspect a recent log file**

```bash
LOG=$(ls -t ~/.local/state/gearup/logs/ | head -1)
head -20 ~/.local/state/gearup/logs/"$LOG"
```

Expected: first lines are the gearup header (`# gearup run log`, `# recipe: Backend`, `# started: ...`), followed by captured command output.

---

## Phase 4A completion criteria

- [ ] `go test ./...` passes across all packages.
- [ ] `gearup version` prints `gearup 0.0.5-phase4a`.
- [ ] `gearup run --recipe X` produces clean output (no brew-list flood).
- [ ] `gearup plan` / `gearup run --dry-run` exits 0 if fully provisioned, 10 otherwise.
- [ ] `gearup run --recipe X --yes` skips the Huh confirm.
- [ ] `gearup run` with piped stdin errors out with exit 3 (TTY guard).
- [ ] Log file exists at `$XDG_STATE_HOME/gearup/logs/...` after a run.
- [ ] No proprietary content anywhere.

---

## Self-review

**Spec coverage:**
- ✅ Silent check stdout / log file — Tasks 1–3 + wiring in Task 6
- ✅ `--dry-run` / `gearup plan` — Tasks 5, 6
- ✅ Exit code 10 for pending work — Tasks 5, 6
- ✅ `--yes` (AutoApprovePrompter) — Tasks 4, 6
- ✅ TTY guard — Task 6

**Type consistency:** `exec.FakeCall` (new), `exec.Runner` (gains `RunQuiet`), `log.File`, `runner.ErrDryRunPending`, `elevation.AutoApprovePrompter` — all referenced consistently across tasks.

**Placeholder scan:** None. Every task contains complete code.

**Trade-offs intentionally deferred:**
- Plan preview is still plain text; Lip Gloss table lands in Phase 4B.
- Huh recipe picker (no `--recipe`) deferred to Phase 4B.
- Spinner during execution deferred to Phase 4B.
- Log rotation not implemented (spec §4.7 calls it out as deferred beyond Phase 4).
