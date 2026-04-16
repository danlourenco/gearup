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
