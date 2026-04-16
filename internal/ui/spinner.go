package ui

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	successIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	failIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗")
	spinChars   = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
)

// StepPrinter renders step-execution status lines. It writes a placeholder
// line per step, then overwrites it with the final status using ANSI escape
// codes. Completed steps remain visible above the current step.
type StepPrinter struct {
	out io.Writer
}

// NewStepPrinter creates a StepPrinter that writes to out.
func NewStepPrinter(out io.Writer) *StepPrinter {
	return &StepPrinter{out: out}
}

// StartCheck prints the checking line for a step.
func (p *StepPrinter) StartCheck(idx, total int, name string) {
	fmt.Fprintf(p.out, "  %s [%d/%d] %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("○"),
		idx, total, name)
}

// FinishSkip overwrites the current step with a dimmed skip line.
func (p *StepPrinter) FinishSkip(idx, total int, name string) {
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		successIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("already installed"))
}

// StartInstall overwrites the checking line with an installing indicator.
func (p *StepPrinter) StartInstall(idx, total int, name string) {
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(string(spinChars[0])),
		idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("installing..."))
}

// FinishInstall overwrites the current step with a success line + elapsed time.
func (p *StepPrinter) FinishInstall(idx, total int, name string, elapsed time.Duration) {
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		successIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(
			fmt.Sprintf("installed (%s)", elapsed.Round(100*time.Millisecond))))
}

// FinishError overwrites the current step with a failure line.
func (p *StepPrinter) FinishError(idx, total int, name, errMsg string) {
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		failIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("FAILED: "+errMsg))
}
