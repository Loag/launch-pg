package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// column is a table column. Width 0 means "take the remaining space".
type column struct {
	title string
	width int
	right bool
}

// table renders rows with a header, a highlighted selection and scrolling.
type table struct {
	columns []column
	rows    [][]string
	// rowStyle optionally colors a whole unselected row.
	rowStyle func(i int) lipgloss.Style
}

func (t table) render(selected, width, height int) string {
	widths := t.widths(width)
	var b strings.Builder
	// Indent by the width of the row selection marker so titles line up.
	b.WriteString("  " + columnStyle.Render(t.line(t.headers(), widths)) + "\n")

	visible := height - 1
	if visible < 1 {
		visible = 1
	}
	start := 0
	if selected >= visible {
		start = selected - visible + 1
	}
	for i := start; i < len(t.rows) && i < start+visible; i++ {
		line := t.line(t.rows[i], widths)
		switch {
		case i == selected:
			line = selectedStyle.Width(width).Render("› " + line)
		case t.rowStyle != nil:
			line = t.rowStyle(i).Render("  " + line)
		default:
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	if len(t.rows) > visible {
		b.WriteString(faintStyle.Render(strings.Repeat(" ", 2) + scrollNote(start, visible, len(t.rows))))
	}
	return b.String()
}

func (t table) headers() []string {
	h := make([]string, len(t.columns))
	for i, c := range t.columns {
		h[i] = c.title
	}
	return h
}

// widths resolves flexible columns against the available width (minus the
// 2-character selection marker and 1-space gaps).
func (t table) widths(total int) []int {
	out := make([]int, len(t.columns))
	fixed, flex := 2, 0
	for i, c := range t.columns {
		out[i] = c.width
		fixed += c.width
		if i > 0 {
			fixed++
		}
		if c.width == 0 {
			flex++
		}
	}
	if flex > 0 {
		each := (total - fixed) / flex
		if each < 8 {
			each = 8
		}
		for i, c := range t.columns {
			if c.width == 0 {
				out[i] = each
			}
		}
	}
	return out
}

func (t table) line(cells []string, widths []int) string {
	parts := make([]string, len(t.columns))
	for i, c := range t.columns {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts[i] = pad(truncate(cell, widths[i]), widths[i], c.right)
	}
	return strings.Join(parts, " ")
}

func truncate(s string, width int) string {
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	if width <= 1 {
		return string(r[:width])
	}
	return string(r[:width-1]) + "…"
}

func pad(s string, width int, right bool) string {
	gap := width - len([]rune(s))
	if gap <= 0 {
		return s
	}
	if right {
		return strings.Repeat(" ", gap) + s
	}
	return s + strings.Repeat(" ", gap)
}

func scrollNote(start, visible, total int) string {
	end := start + visible
	if end > total {
		end = total
	}
	return "showing " + itoa(start+1) + "–" + itoa(end) + " of " + itoa(total)
}
