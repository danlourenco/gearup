// Package runner executes a ResolvedPlan by dispatching each step to its
// registered installer. It stops on the first error (fail-fast). When the
// plan's Recipe declares an Elevation block and at least one step requires
// elevation, the runner batches elevation-required steps after acquiring
// confirmation from the user.
package runner

import (
	"context"
	"fmt"
	"time"

	"gearup/internal/config"
	"gearup/internal/elevation"
	"gearup/internal/installer"
)

// Writer is the minimal output surface the Runner needs.
type Writer interface {
	Printf(format string, args ...any)
	Write(p []byte) (int, error)
}

// Runner orchestrates plan execution.
type Runner struct {
	Registry installer.Registry
	Out      Writer
	Prompter elevation.Prompter // required if any step may require elevation
}

// expiryWarnThreshold is how soon before the elevation window ends we
// print a "running low" warning.
const expiryWarnThreshold = 30 * time.Second

// Run walks plan.Steps. If the Recipe has an Elevation block AND at least
// one step has RequiresElevation:true, elevation is acquired once up front,
// then elevation-required steps run back-to-back, then the remaining steps.
// Otherwise all steps run in declared order.
func (r *Runner) Run(ctx context.Context, plan *config.ResolvedPlan) error {
	elevSteps, regSteps := partition(plan)

	openWindow := len(elevSteps) > 0 &&
		plan.Recipe != nil &&
		plan.Recipe.Elevation != nil

	if openWindow {
		timer, err := elevation.Acquire(ctx, plan.Recipe.Elevation, r.Prompter, r)
		if err != nil {
			return err
		}
		for _, ix := range elevSteps {
			if timer.IsNearExpiry(expiryWarnThreshold) {
				r.Out.Printf("⚠  less than %s remain in elevation window\n", expiryWarnThreshold)
			}
			if err := r.runStep(ctx, ix.idx, ix.step, len(plan.Steps)); err != nil {
				return err
			}
		}
		for _, ix := range regSteps {
			if err := r.runStep(ctx, ix.idx, ix.step, len(plan.Steps)); err != nil {
				return err
			}
		}
		return nil
	}

	for i, step := range plan.Steps {
		if err := r.runStep(ctx, i, step, len(plan.Steps)); err != nil {
			return err
		}
	}
	return nil
}

// Write satisfies io.Writer so elevation.Acquire can print its banner.
// It delegates to r.Out if it satisfies io.Writer (which bufWriter and
// stdPrinter do), otherwise falls back to Printf.
func (r *Runner) Write(p []byte) (int, error) {
	if w, ok := r.Out.(interface {
		Write(p []byte) (int, error)
	}); ok {
		return w.Write(p)
	}
	r.Out.Printf("%s", p)
	return len(p), nil
}

type indexedStep struct {
	idx  int
	step config.Step
}

// partition splits plan.Steps into elevation-required and regular groups,
// preserving declared order within each group. Indexes are 0-based and
// preserved for consistent [i/total] display.
func partition(plan *config.ResolvedPlan) (elev, reg []indexedStep) {
	for i, s := range plan.Steps {
		if s.RequiresElevation {
			elev = append(elev, indexedStep{idx: i, step: s})
		} else {
			reg = append(reg, indexedStep{idx: i, step: s})
		}
	}
	return
}

// runStep executes one step against its registered installer.
func (r *Runner) runStep(ctx context.Context, i int, step config.Step, total int) error {
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
		return nil
	}
	r.Out.Printf("[%d/%d] %s: installing...\n", idx, total, step.Name)
	if err := inst.Install(ctx, step); err != nil {
		return fmt.Errorf("step %d (%s) install failed: %w", idx, step.Name, err)
	}
	r.Out.Printf("[%d/%d] %s: installed\n", idx, total, step.Name)
	return nil
}
