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
