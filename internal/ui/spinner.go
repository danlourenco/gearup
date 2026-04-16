package ui

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	successIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	failIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗")
	spinChars   = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	installStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

// StepPrinter renders step-execution status lines with an animated spinner
// during installs. Completed steps stay as static lines above the current step.
type StepPrinter struct {
	out    io.Writer
	mu     sync.Mutex
	cancel chan struct{} // non-nil while spinner is animating
}

// NewStepPrinter creates a StepPrinter that writes to out.
func NewStepPrinter(out io.Writer) *StepPrinter {
	return &StepPrinter{out: out}
}

// StartCheck prints the checking placeholder line for a step.
func (p *StepPrinter) StartCheck(idx, total int, name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.out, "  %s [%d/%d] %s\n",
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("○"),
		idx, total, name)
}

// FinishSkip overwrites the current line with a dimmed skip status.
func (p *StepPrinter) FinishSkip(idx, total int, name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		successIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("already installed"))
}

// StartInstall begins an animated spinner on the current line.
// The spinner runs in a background goroutine until FinishInstall or FinishError is called.
func (p *StepPrinter) StartInstall(idx, total int, name string) {
	p.mu.Lock()
	p.cancel = make(chan struct{})
	ch := p.cancel
	start := time.Now()
	p.mu.Unlock()

	// Overwrite the "checking" line with the first spinner frame.
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		installStyle.Render(spinChars[0]),
		idx, total, name,
		installStyle.Render("installing..."))

	go func() {
		tick := time.NewTicker(80 * time.Millisecond)
		defer tick.Stop()
		frame := 1
		for {
			select {
			case <-ch:
				return
			case <-tick.C:
				elapsed := time.Since(start).Round(100 * time.Millisecond)
				p.mu.Lock()
				fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
					installStyle.Render(spinChars[frame%len(spinChars)]),
					idx, total, name,
					installStyle.Render(fmt.Sprintf("installing... %s", elapsed)))
				p.mu.Unlock()
				frame++
			}
		}
	}()
}

// stopSpinner stops the background animation goroutine if running.
func (p *StepPrinter) stopSpinner() {
	p.mu.Lock()
	if p.cancel != nil {
		close(p.cancel)
		p.cancel = nil
	}
	p.mu.Unlock()
	// Small sleep to let the goroutine exit and avoid a race on the terminal line.
	time.Sleep(10 * time.Millisecond)
}

// FinishInstall stops the spinner and overwrites with a success line + elapsed time.
func (p *StepPrinter) FinishInstall(idx, total int, name string, elapsed time.Duration) {
	p.stopSpinner()
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		successIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(
			fmt.Sprintf("installed (%s)", elapsed.Round(100*time.Millisecond))))
}

// FinishError stops the spinner and overwrites with a failure line.
func (p *StepPrinter) FinishError(idx, total int, name, errMsg string) {
	p.stopSpinner()
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.out, "\033[1A\033[2K  %s [%d/%d] %s  %s\n",
		failIcon, idx, total, name,
		lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("FAILED: "+errMsg))
}
