package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// confirmScreen asks yes/no before running an action that doesn't produce
// a plan (e.g. removing a server from launch-pg's config).
type confirmScreen struct {
	env    *env
	title  string
	prompt string
	action func() error
	busy   bool
	err    error
}

func newConfirm(e *env, title, prompt string, action func() error) *confirmScreen {
	return &confirmScreen{env: e, title: title, prompt: prompt, action: action}
}

func (s *confirmScreen) Title() string { return s.title }
func (s *confirmScreen) Init() tea.Cmd { return nil }

func (s *confirmScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		s.busy, s.err = false, msg.err
	case tea.KeyMsg:
		if s.busy {
			return s, nil
		}
		switch msg.String() {
		case "y":
			s.busy, s.err = true, nil
			return s, func() tea.Msg {
				if err := s.action(); err != nil {
					return errMsg{err}
				}
				return popMsg{}
			}
		case "n", "esc", "q":
			return s, pop
		}
	}
	return s, nil
}

func (s *confirmScreen) View() string {
	var b strings.Builder
	b.WriteString(warnStyle.Render("⚠ ") + s.prompt + "\n\n")
	switch {
	case s.busy:
		b.WriteString(s.env.loading("Working…"))
	case s.err != nil:
		b.WriteString(errorLine(s.err))
	}
	return b.String()
}

func (s *confirmScreen) Help() string {
	return hint("y", "yes, do it", "n/esc", "cancel")
}
