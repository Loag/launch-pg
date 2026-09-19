package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/templates"
)

type templatesLoadedMsg struct{ list []templates.Template }

// templatesScreen browses built-in and user project templates.
type templatesScreen struct {
	env     *env
	list    []templates.Template
	cur     cursor
	loading bool
	err     error
}

func newTemplatesScreen(e *env) *templatesScreen {
	return &templatesScreen{env: e, loading: true}
}

func (s *templatesScreen) Title() string { return "templates" }

func (s *templatesScreen) Init() tea.Cmd {
	return func() tea.Msg {
		list, err := s.env.svc.Templates.List()
		if err != nil {
			return errMsg{err}
		}
		return templatesLoadedMsg{list}
	}
}

func (s *templatesScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case templatesLoadedMsg:
		s.loading, s.list = false, msg.list
		s.cur.clamp(len(s.list))
	case errMsg:
		s.loading, s.err = false, msg.err
	case tea.KeyMsg:
		if s.cur.move(msg.String(), len(s.list)) {
			return s, nil
		}
		if msg.String() == "esc" || msg.String() == "q" || msg.String() == "left" {
			return s, pop
		}
	}
	return s, nil
}

func (s *templatesScreen) View() string {
	var b strings.Builder
	switch {
	case s.loading:
		b.WriteString(s.env.loading("Loading templates…"))
	case len(s.list) > 0:
		rows := make([][]string, len(s.list))
		for i, t := range s.list {
			rows[i] = []string{t.Name, t.Description}
		}
		t := table{columns: []column{{title: "TEMPLATE", width: 16}, {title: "DESCRIPTION"}}, rows: rows}
		b.WriteString(t.render(s.cur.pos, s.env.bodyWidth(), len(s.list)+1))
		b.WriteString("\n" + s.list[s.cur.pos].Describe())
		b.WriteString("\n" + faintStyle.Render("Add your own in ~/.config/launch-pg/templates/*.yaml; same name overrides a built-in."))
	}
	if s.err != nil {
		b.WriteString("\n\n" + errorLine(s.err))
	}
	return b.String()
}

func (s *templatesScreen) Help() string {
	return hint("↑/↓", "browse", "esc", "back")
}
