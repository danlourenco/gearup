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
