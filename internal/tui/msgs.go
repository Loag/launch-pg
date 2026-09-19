package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/export"
	"launch-pg/internal/service"
)

// Navigation messages, handled by the app model.
type (
	pushMsg    struct{ s screen }
	replaceMsg struct{ s screen }
	popMsg     struct{}
	// resumeMsg is sent to a screen when it becomes visible again, so it
	// can refresh after a change was applied.
	resumeMsg struct{}
)

func push(s screen) tea.Cmd    { return func() tea.Msg { return pushMsg{s} } }
func replace(s screen) tea.Cmd { return func() tea.Msg { return replaceMsg{s} } }
func pop() tea.Msg             { return popMsg{} }

// Operation results.
type (
	errMsg      struct{ err error }
	preparedMsg struct {
		prepared service.Prepared
		targets  []export.Target
	}
	// showMsg asks to display a text result.
	showMsg struct{ title, body string }
)
