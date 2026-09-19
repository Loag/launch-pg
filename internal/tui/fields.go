package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// field is one input in a form. ↑/↓/tab always move between fields, so no
// field may use them; value changes use typing, space or ←/→.
type field interface {
	label() string
	hint() string
	focus() tea.Cmd
	blur()
	update(msg tea.Msg) tea.Cmd
	view(focused bool) string
	value() string
}

// ---- text ----

type textField struct {
	lbl, hnt string
	in       textinput.Model
}

func textInput(label, value, placeholder, hint string) *textField {
	in := textinput.New()
	in.SetValue(value)
	in.Placeholder = placeholder
	in.Prompt = ""
	return &textField{lbl: label, hnt: hint, in: in}
}

func secretInput(label, hint string) *textField {
	f := textInput(label, "", "", hint)
	f.in.EchoMode = textinput.EchoPassword
	f.in.EchoCharacter = '•'
	return f
}

func (f *textField) label() string  { return f.lbl }
func (f *textField) hint() string   { return f.hnt }
func (f *textField) focus() tea.Cmd { return f.in.Focus() }
func (f *textField) blur()          { f.in.Blur() }
func (f *textField) value() string  { return strings.TrimSpace(f.in.Value()) }

func (f *textField) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.in, cmd = f.in.Update(msg)
	return cmd
}

func (f *textField) view(focused bool) string {
	box := "[ " + f.in.View() + " ]"
	if focused {
		return accentStyle.Render(box)
	}
	return box
}

// ---- toggle ----

type toggleField struct {
	lbl, hnt string
	on       bool
}

func toggle(label string, on bool, hint string) *toggleField {
	return &toggleField{lbl: label, hnt: hint, on: on}
}

func (f *toggleField) label() string  { return f.lbl }
func (f *toggleField) hint() string   { return f.hnt }
func (f *toggleField) focus() tea.Cmd { return nil }
func (f *toggleField) blur()          {}

func (f *toggleField) value() string {
	if f.on {
		return "y"
	}
	return ""
}

func (f *toggleField) update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case " ", "left", "right", "h", "l":
			f.on = !f.on
		case "y":
			f.on = true
		case "n":
			f.on = false
		}
	}
	return nil
}

func (f *toggleField) view(focused bool) string {
	yes, no := "  Yes  ", "  No  "
	if f.on {
		yes = selectedStyle.Render("● Yes ")
	} else {
		no = selectedStyle.Render("● No ")
	}
	out := yes + "  " + no
	if focused {
		out += faintStyle.Render("   ←/→ or space to switch")
	}
	return out
}

// ---- choice ----

// option is one choice; label is shown, value is submitted.
type option struct {
	value, label, desc string
}

// choiceField picks one option, changed with ←/→ (wrapping around).
type choiceField struct {
	lbl, hnt string
	opts     []option
	pos      int
}

// choice builds a pick-one field with selected preselected (if present).
func choice(label string, opts []option, selected, hint string) *choiceField {
	f := &choiceField{lbl: label, hnt: hint, opts: opts}
	for i, o := range opts {
		if o.value == selected {
			f.pos = i
		}
	}
	return f
}

func (f *choiceField) label() string  { return f.lbl }
func (f *choiceField) focus() tea.Cmd { return nil }
func (f *choiceField) blur()          {}

func (f *choiceField) hint() string {
	if len(f.opts) > 0 && f.opts[f.pos].desc != "" {
		return f.opts[f.pos].desc
	}
	return f.hnt
}

func (f *choiceField) value() string {
	if len(f.opts) == 0 {
		return ""
	}
	return f.opts[f.pos].value
}

func (f *choiceField) update(msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyMsg)
	if !ok || len(f.opts) == 0 {
		return nil
	}
	n := len(f.opts)
	switch k.String() {
	case "right", "l", " ":
		f.pos = (f.pos + 1) % n
	case "left", "h":
		f.pos = (f.pos - 1 + n) % n
	}
	return nil
}

func (f *choiceField) view(focused bool) string {
	if len(f.opts) == 0 {
		return faintStyle.Render("(nothing to choose from)")
	}
	current := "‹ " + f.opts[f.pos].label + " ›"
	if !focused {
		return current
	}
	return selectedStyle.Render(current) +
		faintStyle.Render("   ←/→ to change · "+itoa(f.pos+1)+" of "+itoa(len(f.opts)))
}

// ---- button ----

// buttonField is the form's submit button; it has no value.
type buttonField struct{ lbl string }

func (f *buttonField) label() string          { return "" }
func (f *buttonField) hint() string           { return "" }
func (f *buttonField) focus() tea.Cmd         { return nil }
func (f *buttonField) blur()                  {}
func (f *buttonField) update(tea.Msg) tea.Cmd { return nil }
func (f *buttonField) value() string          { return "" }

func (f *buttonField) view(focused bool) string {
	if focused {
		return selectedStyle.Render("[ " + f.lbl + " ]")
	}
	return faintStyle.Render("[ " + f.lbl + " ]")
}
