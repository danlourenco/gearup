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
	res, err := i.Runner.RunQuiet(ctx, step.Check)
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
