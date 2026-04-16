// Package runner executes a ResolvedPlan by dispatching each step to its
// registered installer. It stops on the first error (fail-fast).
package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gearup/internal/config"
	"gearup/internal/elevation"
	"gearup/internal/installer"
	"gearup/internal/ui"
)

// ErrDryRunPending is returned by Run when DryRun is true and at least one
// step reports it would be installed.
var ErrDryRunPending = errors.New("dry-run: one or more steps would install")

// Writer is the output surface for non-step messages (banner, summary).
type Writer interface {
	Printf(format string, args ...any)
	Write(p []byte) (int, error)
}

// Runner orchestrates plan execution.
type Runner struct {
	Registry installer.Registry
	Out      Writer
	Prompter elevation.Prompter
	Printer  *ui.StepPrinter // if nil, falls back to plain Printf on Out
	DryRun   bool
}

const expiryWarnThreshold = 30 * time.Second

// Run walks plan.Steps.
func (r *Runner) Run(ctx context.Context, plan *config.ResolvedPlan) error {
	if r.DryRun {
		return r.runDryRun(ctx, plan)
	}
	return r.runLive(ctx, plan)
}

func (r *Runner) runDryRun(ctx context.Context, plan *config.ResolvedPlan) error {
	total := len(plan.Steps)
	var statuses []ui.StepStatus
	willInstall := 0

	for i, step := range plan.Steps {
		idx := i + 1
		inst, err := r.Registry.Get(step.Type)
		if err != nil {
			return fmt.Errorf("step %d (%s): %w", idx, step.Name, err)
		}
		installed, err := inst.Check(ctx, step)
		if err != nil {
			return fmt.Errorf("step %d (%s) check failed: %w", idx, step.Name, err)
		}
		statuses = append(statuses, ui.StepStatus{
			Index:             idx,
			Total:             total,
			Name:              step.Name,
			Installed:         installed,
			RequiresElevation: step.RequiresElevation,
		})
		if !installed {
			willInstall++
		}
	}

	r.Out.Printf("%s", ui.RenderPreview(statuses))

	if willInstall > 0 {
		return ErrDryRunPending
	}
	return nil
}

func (r *Runner) runLive(ctx context.Context, plan *config.ResolvedPlan) error {
	elevSteps, regSteps := partition(plan)

	// Pre-check: do any elevation-required steps actually need to run?
	needsElevation := false
	if len(elevSteps) > 0 && plan.Config != nil && plan.Config.Elevation != nil {
		for _, ix := range elevSteps {
			inst, err := r.Registry.Get(ix.step.Type)
			if err != nil {
				return fmt.Errorf("step %d (%s): %w", ix.idx+1, ix.step.Name, err)
			}
			installed, err := inst.Check(ctx, ix.step)
			if err != nil {
				return fmt.Errorf("step %d (%s) check failed: %w", ix.idx+1, ix.step.Name, err)
			}
			if !installed {
				needsElevation = true
				break
			}
		}
	}

	if needsElevation {
		timer, err := elevation.Acquire(ctx, plan.Config.Elevation, r.Prompter, r)
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

	// No elevation needed — run in declared order.
	for i, step := range plan.Steps {
		if err := r.runStep(ctx, i, step, len(plan.Steps)); err != nil {
			return err
		}
	}
	return nil
}

// Write satisfies io.Writer so elevation.Acquire can print its banner.
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

func (r *Runner) runStep(ctx context.Context, i int, step config.Step, total int) error {
	idx := i + 1
	inst, err := r.Registry.Get(step.Type)
	if err != nil {
		return fmt.Errorf("step %d (%s): %w", idx, step.Name, err)
	}

	if r.Printer != nil {
		r.Printer.StartCheck(idx, total, step.Name)
	} else {
		r.Out.Printf("[%d/%d] %s: checking...\n", idx, total, step.Name)
	}

	installed, err := inst.Check(ctx, step)
	if err != nil {
		if r.Printer != nil {
			r.Printer.FinishError(idx, total, step.Name, err.Error())
		}
		return fmt.Errorf("step %d (%s) check failed: %w", idx, step.Name, err)
	}
	if installed {
		if r.Printer != nil {
			r.Printer.FinishSkip(idx, total, step.Name)
		} else {
			r.Out.Printf("[%d/%d] %s: already installed — skip\n", idx, total, step.Name)
		}
		return nil
	}

	start := time.Now()
	if r.Printer != nil {
		r.Printer.StartInstall(idx, total, step.Name)
	} else {
		r.Out.Printf("[%d/%d] %s: installing...\n", idx, total, step.Name)
	}

	if err := inst.Install(ctx, step); err != nil {
		if r.Printer != nil {
			r.Printer.FinishError(idx, total, step.Name, err.Error())
		}
		return fmt.Errorf("step %d (%s) install failed: %w", idx, step.Name, err)
	}

	if r.Printer != nil {
		r.Printer.FinishInstall(idx, total, step.Name, time.Since(start))
	} else {
		r.Out.Printf("[%d/%d] %s: installed\n", idx, total, step.Name)
	}
	return nil
}
