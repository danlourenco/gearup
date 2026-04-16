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

// Check runs the step's Check command if set, otherwise `brew list --formula <name>`.
// Exit 0 means installed. Recipes can override the default check to handle
// tools that may already be installed from other sources (e.g. `check: command -v git`)
// or to work around brew-alias formula names.
func (i *Installer) Check(ctx context.Context, step config.Step) (bool, error) {
	if step.Formula == "" {
		return false, fmt.Errorf("brew step %q missing formula", step.Name)
	}
	cmd := step.Check
	if cmd == "" {
		cmd = fmt.Sprintf("brew list --formula %s", step.Formula)
	}
	res, err := i.Runner.RunQuiet(ctx, cmd)
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
