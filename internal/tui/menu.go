package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// menuScreen lists actions grouped under headings, with the highlighted
// action's description underneath.
type menuScreen struct {
	env    *env
	title  string
	groups []actionGroup
	flat   []action
	cur    cursor
	busy   bool
	err    error
}

func newMenu(e *env, title string, groups []actionGroup) *menuScreen {
	m := &menuScreen{env: e, title: title, groups: groups}
	for _, g := range groups {
		m.flat = append(m.flat, g.actions...)
	}
	return m
}

func (m *menuScreen) Title() string { return m.title }
func (m *menuScreen) Init() tea.Cmd { return nil }

func (m *menuScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	if cmd, ok := routeResult(m.env, msg); ok {
		return m, cmd
	}
	switch msg := msg.(type) {
	case errMsg:
		m.busy, m.err = false, msg.err
	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}
		key := msg.String()
		if m.cur.move(key, len(m.flat)) {
			return m, nil
		}
		switch key {
		case "esc", "q":
			return m, pop
		case "enter":
			if len(m.flat) > 0 {
				return m.start(m.flat[m.cur.pos])
			}
		default:
			if a, ok := findAction(key, m.groups); ok {
				return m.start(a)
			}
		}
	}
	return m, nil
}

func (m *menuScreen) start(a action) (screen, tea.Cmd) {
	if a.run != nil {
		m.busy, m.err = true, nil
	}
	return m, a.fromMenu()
}

func (m *menuScreen) View() string {
	var b strings.Builder
	i := 0
	for gi, g := range m.groups {
		if len(g.actions) == 0 {
			continue
		}
		if gi > 0 {
			b.WriteString("\n")
		}
		b.WriteString(sectionStyle.Render(strings.ToUpper(g.title)) + "\n")
		for _, a := range g.actions {
			label := pad(a.label, 34, false) + keyStyle.Render(displayKey(a.key))
			if i == m.cur.pos {
				b.WriteString(selectedStyle.Width(m.env.bodyWidth()).Render("› "+label) + "\n")
			} else {
				b.WriteString("  " + label + "\n")
			}
			i++
		}
	}
	if len(m.flat) > 0 {
		b.WriteString("\n" + faintStyle.Render(m.flat[m.cur.pos].desc) + "\n")
	}
	switch {
	case m.busy:
		b.WriteString("\n" + m.env.loading("Working…"))
	case m.err != nil:
		b.WriteString("\n" + errorLine(m.err))
	}
	return b.String()
}

func (m *menuScreen) Help() string {
	return hint("↑/↓", "choose", "enter", "select", "esc", "back")
}

// routeResult handles the outcome of a started action for screens that
// should get out of the way (menus, forms): a plan or a message replaces
// the current screen.
func routeResult(e *env, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case preparedMsg:
		return replace(newPlanScreen(e, msg.prepared, msg.targets)), true
	case showMsg:
		return replace(newMessageScreen(e, msg.title, msg.body)), true
	}
	return nil, false
}
