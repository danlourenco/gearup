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
