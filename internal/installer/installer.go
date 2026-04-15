// Package installer defines the Installer interface and a Registry mapping
// step types to implementations. Type-specific installers live in sub-packages.
package installer

import (
	"context"
	"fmt"

	"gearup/internal/config"
)

// Installer provides idempotent check and install for one step type.
//
// Check reports whether the step is already satisfied on the host.
// When Check returns (true, nil) the runner skips Install for that step.
//
// Install executes the provisioning action. It must return an error if
// the action failed for any reason.
type Installer interface {
	Check(ctx context.Context, step config.Step) (installed bool, err error)
	Install(ctx context.Context, step config.Step) error
}

// Registry maps step type names (e.g. "brew") to their Installer.
type Registry map[string]Installer

// Get returns the Installer for a step type or an error if none is registered.
func (r Registry) Get(stepType string) (Installer, error) {
	inst, ok := r[stepType]
	if !ok {
		return nil, fmt.Errorf("no installer registered for step type %q", stepType)
	}
	return inst, nil
}
