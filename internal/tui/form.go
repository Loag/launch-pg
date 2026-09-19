package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// formScreen collects field values and runs submit when the user presses
// the button at the bottom. submit's command must produce a preparedMsg
// (→ plan screen), showMsg (→ message) or errMsg (shown on the form), or
// a navigation message.
type formScreen struct {
	env    *env
	title  string
	intro  string
	fields []field
	focus  int
	submit func(values []string) tea.Cmd
	busy   bool
	err    error
}

// newForm builds a form. values passed to submit line up with fields.
func newForm(e *env, title, intro, button string, fields []field, submit func(values []string) tea.Cmd) *formScreen {
	if button == "" {
		button = "Continue"
	}
	return &formScreen{
		env: e, title: title, intro: intro, submit: submit,
		fields: append(fields, &buttonField{lbl: button}),
	}
}

func (f *formScreen) Title() string { return f.title }
func (f *formScreen) Init() tea.Cmd { return f.fields[0].focus() }

func (f *formScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	if cmd, ok := routeResult(f.env, msg); ok {
		return f, cmd
	}
	switch msg := msg.(type) {
	case errMsg:
		f.busy, f.err = false, msg.err
		return f, nil
	case tea.KeyMsg:
		if f.busy {
			return f, nil
		}
		switch msg.String() {
		case "esc":
			return f, pop
		case "tab", "down":
			return f, f.setFocus(f.focus + 1)
		case "shift+tab", "up":
			return f, f.setFocus(f.focus - 1)
		case "enter":
			if _, isButton := f.fields[f.focus].(*buttonField); !isButton {
				return f, f.setFocus(f.focus + 1)
			}
			f.busy, f.err = true, nil
			return f, f.submit(f.values())
		}
	}
	return f, f.fields[f.focus].update(msg)
}

func (f *formScreen) setFocus(i int) tea.Cmd {
	if i < 0 || i >= len(f.fields) {
		return nil
	}
	f.fields[f.focus].blur()
	f.focus = i
	return f.fields[i].focus()
}

// values excludes the trailing button.
func (f *formScreen) values() []string {
	out := make([]string, len(f.fields)-1)
	for i := range out {
		out[i] = f.fields[i].value()
	}
	return out
}

func (f *formScreen) View() string {
	var lines []string
	add := func(s string) { lines = append(lines, strings.Split(s, "\n")...) }

	if f.intro != "" {
		add(lipgloss.NewStyle().Width(f.env.bodyWidth()).Render(faintStyle.Render(f.intro)))
		add("")
	}

	focusStart, focusEnd := 0, 0
	for i, fld := range f.fields {
		focused := i == f.focus
		if focused {
			focusStart = len(lines)
		}
		if lbl := fld.label(); lbl != "" {
			marker := "  "
			if focused {
				marker = accentStyle.Render("› ")
			}
			add(marker + columnStyle.Render(lbl))
		}
		add(indent(fld.view(focused), "  "))
		if focused && fld.hint() != "" {
			add(indent(faintStyle.Render(fld.hint()), "  "))
		}
		if focused {
			focusEnd = len(lines)
		}
		add("")
	}

	status := ""
	switch {
	case f.busy:
		status = f.env.loading("Working…")
	case f.err != nil:
		status = errorLine(f.err)
	}
	height := f.env.bodyHeight()
	if status != "" {
		height -= 2
	}

	out := strings.Join(scrollWindow(lines, focusStart, focusEnd, height), "\n")
	if status != "" {
		out += "\n\n" + status
	}
	return out
}

// scrollWindow returns at most height lines of lines, keeping the focused
// block [focusStart, focusEnd) visible and marking hidden content.
func scrollWindow(lines []string, focusStart, focusEnd, height int) []string {
	if len(lines) <= height {
		return lines
	}
	room := max(height-2, 1) // leave space for the more-above/below markers
	start := 0
	if focusEnd > room {
		start = focusEnd - room
	}
	if focusStart < start {
		start = focusStart
	}
	end := min(start+room, len(lines))

	out := make([]string, 0, height)
	if start > 0 {
		out = append(out, faintStyle.Render("  ↑ more above"))
	} else {
		out = append(out, "")
	}
	out = append(out, lines[start:end]...)
	if end < len(lines) {
		out = append(out, faintStyle.Render("  ↓ more below"))
	}
	return out
}

func (f *formScreen) Help() string {
	switch f.fields[f.focus].(type) {
	case *choiceField, *toggleField:
		return hint("←/→", "change", "↑/↓", "move", "enter", "next", "esc", "cancel")
	case *buttonField:
		return hint("enter", "submit", "↑", "back to fields", "esc", "cancel")
	}
	return hint("↑/↓", "move", "enter", "next", "esc", "cancel")
}

func indent(s, prefix string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}
