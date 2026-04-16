// Package brewcask implements the "brew-cask" step type for GUI applications.
package brewcask

import (
	"context"
	"fmt"

	"gearup/internal/config"
	gearexec "gearup/internal/exec"
)

// Installer is the brew-cask step installer.
type Installer struct {
	Runner gearexec.Runner
}

// Check runs the step's Check command if set, otherwise `brew list --cask <name>`.
func (i *Installer) Check(ctx context.Context, step config.Step) (bool, error) {
	if step.Cask == "" {
		return false, fmt.Errorf("brew-cask step %q missing cask", step.Name)
	}
	cmd := step.Check
	if cmd == "" {
		cmd = fmt.Sprintf("brew list --cask %s", step.Cask)
	}
	res, err := i.Runner.RunQuiet(ctx, cmd)
	if err != nil {
		return false, err
	}
	return res.ExitCode == 0, nil
}

// Install runs `brew install --cask <cask>` and fails on non-zero exit.
func (i *Installer) Install(ctx context.Context, step config.Step) error {
	if step.Cask == "" {
		return fmt.Errorf("brew-cask step %q missing cask", step.Name)
	}
	cmd := fmt.Sprintf("brew install --cask %s", step.Cask)
	res, err := i.Runner.RunQuiet(ctx, cmd)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("brew install --cask %s failed (exit %d): %s", step.Cask, res.ExitCode, res.Stderr)
	}
	return nil
}
