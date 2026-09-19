package tui

import "github.com/charmbracelet/lipgloss"

// Palette adapts to light and dark terminals.
var (
	colorAccent  = lipgloss.AdaptiveColor{Light: "#5A3FC0", Dark: "#A594FF"}
	colorHeading = lipgloss.AdaptiveColor{Light: "#00796B", Dark: "#4FD6C4"}
	colorSubtle  = lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#6C6C6C"}
	colorBorder  = lipgloss.AdaptiveColor{Light: "#C9C2EE", Dark: "#4A4270"}
	colorSelect  = lipgloss.AdaptiveColor{Light: "#E6E0FF", Dark: "#3B3363"}
	colorError   = lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#FF7A7A"}
	colorWarn    = lipgloss.AdaptiveColor{Light: "#A86A00", Dark: "#F6C85F"}
	colorOK      = lipgloss.AdaptiveColor{Light: "#2F855A", Dark: "#6BD49B"}
	colorHeaderF = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"}
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(colorHeaderF).Background(colorAccent).Padding(0, 1)
	crumbStyle  = lipgloss.NewStyle().Foreground(colorHeaderF).Background(colorAccent)
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(0, 1)

	titleStyle    = lipgloss.NewStyle().Bold(true)
	accentStyle   = lipgloss.NewStyle().Foreground(colorAccent)
	columnStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorHeading)
	labelStyle    = lipgloss.NewStyle().Foreground(colorHeading)
	selectedStyle = lipgloss.NewStyle().Background(colorSelect).Bold(true)
	faintStyle    = lipgloss.NewStyle().Foreground(colorSubtle)
	sectionStyle  = lipgloss.NewStyle().Foreground(colorHeading).Bold(true)
	keyStyle      = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	errorStyle = lipgloss.NewStyle().Foreground(colorError)
	warnStyle  = lipgloss.NewStyle().Foreground(colorWarn)
	okStyle    = lipgloss.NewStyle().Foreground(colorOK)
)

// hint renders "key label" pairs for the footer.
func hint(pairs ...string) string {
	out := ""
	for i := 0; i+1 < len(pairs); i += 2 {
		if out != "" {
			out += faintStyle.Render("  ·  ")
		}
		out += keyStyle.Render(pairs[i]) + " " + faintStyle.Render(pairs[i+1])
	}
	return out
}

func errorLine(err error) string {
	return errorStyle.Render("✗ " + err.Error())
}
