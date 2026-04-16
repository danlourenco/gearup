package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StepStatus holds the pre-check result for one step, used by the plan preview.
type StepStatus struct {
	Index             int
	Total             int
	Name              string
	Installed         bool
	RequiresElevation bool
}

var (
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	arrow     = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("→")
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	boldStyle = lipgloss.NewStyle().Bold(true)
	elevTag   = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("[elevation]")
)

// RenderPreview returns a styled multi-line plan preview string.
func RenderPreview(steps []StepStatus) string {
	var b strings.Builder
	toInstall := 0
	needsElev := 0

	for _, s := range steps {
		prefix := fmt.Sprintf("[%d/%d]", s.Index, s.Total)
		if s.Installed {
			fmt.Fprintf(&b, "  %s %s %s  %s\n",
				dimStyle.Render(prefix),
				checkMark,
				s.Name,
				dimStyle.Render("already installed"),
			)
		} else {
			toInstall++
			elev := ""
			if s.RequiresElevation {
				needsElev++
				elev = "  " + elevTag
			}
			fmt.Fprintf(&b, "  %s %s %s  %s%s\n",
				prefix,
				arrow,
				boldStyle.Render(s.Name),
				"will install",
				elev,
			)
		}
	}

	total := len(steps)
	summary := fmt.Sprintf("\n  %d to install · %d already installed",
		toInstall, total-toInstall)
	if needsElev > 0 {
		summary += fmt.Sprintf(" · %d requires elevation", needsElev)
	}
	b.WriteString(summary + "\n")

	return b.String()
}
