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
