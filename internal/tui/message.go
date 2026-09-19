package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// messageScreen shows scrollable text until dismissed.
type messageScreen struct {
	env   *env
	title string
	view  viewport.Model
}

func newMessageScreen(e *env, title, body string) *messageScreen {
	vp := viewport.New(e.bodyWidth(), e.bodyHeight())
	vp.SetContent(body)
	return &messageScreen{env: e, title: title, view: vp}
}

func (s *messageScreen) Title() string { return s.title }
func (s *messageScreen) Init() tea.Cmd { return nil }

func (s *messageScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.view.Width, s.view.Height = s.env.bodyWidth(), s.env.bodyHeight()
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "enter", "q":
			return s, pop
		}
	}
	var cmd tea.Cmd
	s.view, cmd = s.view.Update(msg)
	return s, cmd
}

func (s *messageScreen) View() string { return s.view.View() }

func (s *messageScreen) Help() string {
	return hint("↑/↓", "scroll", "enter/esc", "back")
}
