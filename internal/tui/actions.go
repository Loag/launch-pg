package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// action is something the user can do. It is defined once and offered both
// as a shortcut key on a list and as an entry in that list's action menu.
// Exactly one of open and run is set:
//   - open builds a screen (form, confirmation, sub-list)
//   - run starts work that ends in preparedMsg, showMsg or errMsg
type action struct {
	key   string
	label string
	desc  string
	open  func() screen
	run   func() tea.Cmd
}

// actionGroup is a titled set of actions (e.g. "myapp_app" and "General").
type actionGroup struct {
	title   string
	actions []action
}

// fromList starts an action from a list screen: screens are pushed on top
// of the list; work runs in place and the list handles its result.
func (a action) fromList() tea.Cmd {
	if a.open != nil {
		return push(a.open())
	}
	return a.run()
}

// fromMenu starts an action from the action menu, which then gets out of
// the way: screens replace the menu so "back" returns to the list.
func (a action) fromMenu() tea.Cmd {
	if a.open != nil {
		return replace(a.open())
	}
	return a.run()
}

// findAction returns the action bound to key.
func findAction(key string, groups []actionGroup) (action, bool) {
	for _, g := range groups {
		for _, a := range g.actions {
			if a.key == key {
				return a, true
			}
		}
	}
	return action{}, false
}

// handleListKey implements the keys every list screen shares: enter opens
// the action menu, ? opens help, and shortcut keys run their action.
func handleListKey(e *env, key, title string, groups []actionGroup) (tea.Cmd, bool) {
	switch key {
	case "enter":
		return push(newMenu(e, title, groups)), true
	case "?":
		return push(newMessageScreen(e, "help", helpText(groups))), true
	}
	if a, ok := findAction(key, groups); ok {
		return a.fromList(), true
	}
	return nil, false
}

// helpText lists every action and the navigation keys.
func helpText(groups []actionGroup) string {
	var b strings.Builder
	for _, g := range groups {
		if len(g.actions) == 0 {
			continue
		}
		b.WriteString(sectionStyle.Render(strings.ToUpper(g.title)) + "\n")
		for _, a := range g.actions {
			fmt.Fprintf(&b, "  %s  %-26s %s\n", keyStyle.Render(fmt.Sprintf("%-6s", displayKey(a.key))), a.label, faintStyle.Render(a.desc))
		}
		b.WriteString("\n")
	}
	b.WriteString(sectionStyle.Render("NAVIGATION") + "\n")
	for _, kv := range [][2]string{
		{"↑/↓", "move (also k/j, pgup/pgdown, home/end)"},
		{"enter", "open the action menu for the selected row"},
		{"→", "go into the selected row (servers, databases)"},
		{"esc", "back"},
		{"r", "refresh"},
		{"?", "this help"},
		{"ctrl+c", "quit"},
	} {
		fmt.Fprintf(&b, "  %s  %s\n", keyStyle.Render(fmt.Sprintf("%-6s", kv[0])), kv[1])
	}
	return b.String()
}

func itoa(i int) string { return strconv.Itoa(i) }

// displayKey shows key names the way users see them on the keyboard.
func displayKey(k string) string {
	switch k {
	case "right":
		return "→"
	}
	return k
}
