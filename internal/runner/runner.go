// Package runner executes a ResolvedPlan by dispatching each step to its
// registered installer. It stops on the first error (fail-fast).
package runner

import (
	"context"
	"fmt"

	"gearup/internal/config"
	"gearup/internal/installer"
)

// Writer is the minimal output surface the Runner needs. Real use passes
// a stdout wrapper; tests pass a buffer.
type Writer interface {
	Printf(format string, args ...any)
}

// Runner orchestrates plan execution.
type Runner struct {
	Registry installer.Registry
	Out      Writer
}

// Run walks plan.Steps in order. For each step: look up its installer,
// call Check; if already installed, skip; otherwise call Install.
// Returns on the first error.
func (r *Runner) Run(ctx context.Context, plan *config.ResolvedPlan) error {
	total := len(plan.Steps)
	for i, step := range plan.Steps {
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
			continue
		}
		r.Out.Printf("[%d/%d] %s: installing...\n", idx, total, step.Name)
		if err := inst.Install(ctx, step); err != nil {
			return fmt.Errorf("step %d (%s) install failed: %w", idx, step.Name, err)
		}
		r.Out.Printf("[%d/%d] %s: installed\n", idx, total, step.Name)
	}
	return nil
}
