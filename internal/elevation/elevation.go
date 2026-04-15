package elevation

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"gearup/internal/config"
)

// Prompter is the interactive surface Acquire uses to ask the user to
// confirm they have acquired admin permissions.
type Prompter interface {
	// Confirm shows title and returns (true, nil) if the user affirms,
	// (false, nil) if they abort, (_, err) on any other failure.
	Confirm(title string) (bool, error)
}

// HuhPrompter is the production Prompter backed by charmbracelet/huh.
type HuhPrompter struct{}

// Confirm renders a Huh confirm prompt.
func (HuhPrompter) Confirm(title string) (bool, error) {
	var ok bool
	err := huh.NewConfirm().
		Title(title).
		Affirmative("Continue").
		Negative("Abort").
		Value(&ok).
		Run()
	if err != nil {
		return false, err
	}
	return ok, nil
}

// bannerStyle is the Lip Gloss style for elevation banners.
var bannerStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("214")). // amber-yellow
	Padding(1, 2).
	Bold(true)

// Acquire prints a styled banner containing cfg.Message, prompts the user
// for confirmation via the supplied Prompter, and returns a Timer started
// at the confirmation moment. Acquire returns an error if the user aborts
// or the Prompter fails.
func Acquire(_ context.Context, cfg *config.Elevation, p Prompter, out io.Writer) (*Timer, error) {
	if cfg == nil || cfg.Message == "" {
		return nil, fmt.Errorf("elevation: message is required")
	}

	fmt.Fprintln(out, bannerStyle.Render(cfg.Message))

	ok, err := p.Confirm("Proceed with elevation-required steps?")
	if err != nil {
		return nil, fmt.Errorf("elevation confirm: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("elevation aborted by user")
	}

	return NewTimer(cfg.Duration), nil
}
