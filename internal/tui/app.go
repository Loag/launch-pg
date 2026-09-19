package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// appModel owns the navigation stack, draws the frame (header, panel,
// footer) and routes messages to the top screen.
type appModel struct {
	env   *env
	stack []screen
}

func newAppModel(e *env, root screen) appModel {
	e.spinner = spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(accentStyle))
	return appModel{env: e, stack: []screen{root}}
}

func (m appModel) top() screen { return m.stack[len(m.stack)-1] }

func (m appModel) Init() tea.Cmd {
	return tea.Batch(m.env.spinner.Tick, m.top().Init())
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.env.spinner, cmd = m.env.spinner.Update(msg)
		return m, cmd
	case tea.WindowSizeMsg:
		m.env.width, m.env.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case pushMsg:
		m.stack = append(m.stack, msg.s)
		return m, msg.s.Init()
	case replaceMsg:
		m.stack[len(m.stack)-1] = msg.s
		return m, msg.s.Init()
	case popMsg:
		m.stack = m.stack[:len(m.stack)-1]
		if len(m.stack) == 0 {
			return m, tea.Quit
		}
		return m.forward(resumeMsg{})
	}
	return m.forward(msg)
}

func (m appModel) forward(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.top().Update(msg)
	m.stack[len(m.stack)-1] = next
	return m, cmd
}

func (m appModel) View() string {
	width := m.env.width
	if width == 0 {
		return ""
	}
	body := lipgloss.NewStyle().
		Width(m.env.bodyWidth()).
		Height(m.env.bodyHeight()).
		MaxHeight(m.env.bodyHeight()).
		Render(m.top().View())
	return lipgloss.JoinVertical(lipgloss.Left,
		m.header(width),
		panelStyle.Render(body),
		" "+m.top().Help(),
	)
}

func (m appModel) header(width int) string {
	titles := make([]string, len(m.stack))
	for i, s := range m.stack {
		titles[i] = s.Title()
	}
	left := headerStyle.Render("launch-pg") + crumbStyle.Render(" "+strings.Join(titles, " › ")+" ")
	right := crumbStyle.Render(" " + m.env.svc.Version + " ")
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return left + crumbStyle.Render(strings.Repeat(" ", gap)) + right
}
