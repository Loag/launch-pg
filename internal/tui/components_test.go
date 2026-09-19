package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func kindByName(t *testing.T, name string) exportKind {
	t.Helper()
	for _, k := range exportKinds {
		if k.kind == name {
			return k
		}
	}
	t.Fatalf("no export kind %q", name)
	return exportKind{}
}

func TestBuildTargetSkipsEmptyAndAddsDatabase(t *testing.T) {
	k := kindByName(t, "k8s") // namespace, name, out, host
	got, err := buildTarget(k, []string{"apps", "", "k8s/db.yaml", ""}, "myapp")
	if err != nil {
		t.Fatal(err)
	}
	if want := "k8s:namespace=apps,out=k8s/db.yaml,dbname=myapp"; got != want {
		t.Errorf("target = %q, want %q", got, want)
	}
}

func TestBuildTargetValidates(t *testing.T) {
	sealed := kindByName(t, "sealedsecret")
	values := make([]string, len(sealed.fields))
	if _, err := buildTarget(sealed, values, ""); err == nil {
		t.Error("expected error for missing required cert")
	}

	k := kindByName(t, "env")
	if _, err := buildTarget(k, []string{"a,b", "", ""}, ""); err == nil {
		t.Error("expected error for comma in value")
	}
}

func TestFindAction(t *testing.T) {
	groups := []actionGroup{
		{actions: []action{{key: "a", label: "first"}}},
		{actions: []action{{key: "b", label: "second"}}},
	}
	if a, ok := findAction("b", groups); !ok || a.label != "second" {
		t.Errorf("findAction(b) = %+v, %v", a, ok)
	}
	if _, ok := findAction("z", groups); ok {
		t.Error("found an unbound key")
	}
}

func TestChoiceFieldPreselectsAndMoves(t *testing.T) {
	f := choice("Preset", []option{{"owner", "owner", ""}, {"readwrite", "readwrite", ""}, {"readonly", "readonly", ""}}, "readwrite", "")
	if f.value() != "readwrite" {
		t.Errorf("preselected = %q", f.value())
	}
	f.update(tea.KeyMsg{Type: tea.KeyRight})
	if f.value() != "readonly" {
		t.Errorf("after right = %q", f.value())
	}
	f.update(tea.KeyMsg{Type: tea.KeyRight})
	if f.value() != "owner" {
		t.Errorf("right should wrap to the first option, got %q", f.value())
	}
	f.update(tea.KeyMsg{Type: tea.KeyUp})
	if f.value() != "owner" {
		t.Errorf("up must not change a choice, got %q", f.value())
	}
}

// Regression: ↑ on a choice field used to change its value instead of
// moving to the previous field.
func TestFormUpLeavesChoiceField(t *testing.T) {
	f := newForm(&env{}, "t", "", "", []field{
		textInput("a", "", "", ""),
		choice("b", []option{{"x", "x", ""}, {"y", "y", ""}}, "y", ""),
	}, func([]string) tea.Cmd { return nil })
	f.Update(tea.KeyMsg{Type: tea.KeyDown})
	f.Update(tea.KeyMsg{Type: tea.KeyUp})
	if f.focus != 0 {
		t.Errorf("focus = %d, want 0", f.focus)
	}
	if got := f.fields[1].value(); got != "y" {
		t.Errorf("choice changed to %q while navigating", got)
	}
}

func TestScrollWindowKeepsFocusVisible(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = itoa(i)
	}
	got := scrollWindow(lines, 25, 27, 10)
	if len(got) > 10 {
		t.Fatalf("window has %d lines, want ≤ 10", len(got))
	}
	joined := strings.Join(got, "|")
	if !strings.Contains(joined, "|25|26") || !strings.Contains(joined, "more above") {
		t.Errorf("focused lines not visible: %v", got)
	}
}

func TestToggleField(t *testing.T) {
	f := toggle("x", false, "")
	f.update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if !isYes(f.value()) {
		t.Error("space should switch the toggle on")
	}
}

func TestFormSubmitsValuesFromButton(t *testing.T) {
	var got []string
	f := newForm(&env{}, "t", "", "", []field{
		textInput("a", "one", "", ""),
		toggle("b", true, ""),
	}, func(v []string) tea.Cmd {
		got = v
		return nil
	})
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	f.Update(enter) // a → b
	f.Update(enter) // b → button
	f.Update(enter) // submit
	if len(got) != 2 || got[0] != "one" || got[1] != "y" {
		t.Errorf("submitted %v", got)
	}
}

func TestTableWidthsFillSpace(t *testing.T) {
	tb := table{columns: []column{{title: "A", width: 10}, {title: "B"}}}
	w := tb.widths(50)
	// 2 marker + 10 + 1 gap = 13 fixed, so B gets 37.
	if w[1] != 37 {
		t.Errorf("flex width = %d, want 37", w[1])
	}
}
