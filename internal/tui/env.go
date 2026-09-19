package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/app"
)

// Frame overhead: header line, panel border (2), footer line.
const frameLines = 4

// env is shared by all screens: services, context, terminal size and the
// app-wide spinner.
type env struct {
	ctx     context.Context
	svc     app.Services
	width   int
	height  int
	spinner spinner.Model
}

// bodyHeight is the number of lines available inside the panel.
func (e *env) bodyHeight() int {
	if h := e.height - frameLines; h > 5 {
		return h
	}
	return 5
}

// bodyWidth is the usable width inside the panel (border + padding).
func (e *env) bodyWidth() int {
	if w := e.width - 4; w > 20 {
		return w
	}
	return 20
}

func (e *env) loading(what string) string {
	return e.spinner.View() + " " + faintStyle.Render(what)
}

// screen is one page on the navigation stack.
type screen interface {
	Title() string
	Init() tea.Cmd
	Update(msg tea.Msg) (screen, tea.Cmd)
	// View renders the panel body.
	View() string
	// Help renders the footer key hints.
	Help() string
}
