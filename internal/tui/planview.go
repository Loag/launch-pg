package tui

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/export"
	"launch-pg/internal/plan"
	"launch-pg/internal/service"
)

type planState int

const (
	planConfirm planState = iota
	planApplying
	planDone
)

// planScreen shows a prepared plan, applies it on confirmation, and then
// shows the outcome and any new credentials.
type planScreen struct {
	env      *env
	prepared service.Prepared
	targets  []export.Target
	state    planState
	view     viewport.Model
}

func newPlanScreen(e *env, p service.Prepared, targets []export.Target) *planScreen {
	s := &planScreen{env: e, prepared: p, targets: targets, view: viewport.New(e.bodyWidth(), e.bodyHeight())}
	s.view.SetContent(s.planText())
	return s
}

func (s *planScreen) Title() string { return "Plan" }
func (s *planScreen) Init() tea.Cmd { return nil }

func (s *planScreen) planText() string {
	var b bytes.Buffer
	for _, w := range s.prepared.Warnings {
		b.WriteString(warnStyle.Render("⚠ "+w) + "\n")
	}
	if s.prepared.Plan.Empty() {
		b.WriteString(okStyle.Render("✓ Nothing to do — the server already matches.") + "\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%s on %s. Nothing has changed yet; press y to apply.\n",
		titleStyle.Render(fmt.Sprintf("%d change(s)", len(s.prepared.Plan.Actions))),
		accentStyle.Render(s.prepared.Server.Name))
	if bk := s.prepared.Backup; bk != nil {
		fmt.Fprintf(&b, "\n-- 0. back up %s with pg_dump to %s\n", bk.Database, bk.Path)
	}
	if err := plan.Render(&b, s.prepared.Plan, s.prepared.Server.MaintenanceDB); err != nil {
		b.WriteString(errorStyle.Render(err.Error()))
	}
	for _, t := range s.targets {
		fmt.Fprintf(&b, "-- then export new credentials to %s\n", t.Raw)
	}
	return b.String()
}

func (s *planScreen) Update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.view.Width, s.view.Height = s.env.bodyWidth(), s.env.bodyHeight()
	case appliedMsg:
		s.state = planDone
		s.view.SetContent(s.resultText(msg))
		s.view.GotoTop()
		return s, nil
	case tea.KeyMsg:
		switch s.state {
		case planConfirm:
			switch msg.String() {
			case "y":
				if s.prepared.Plan.Empty() {
					return s, pop
				}
				s.state = planApplying
				return s, applyCmd(s.env, s.prepared, s.targets)
			case "n", "esc", "q":
				return s, pop
			}
		case planApplying:
			return s, nil
		case planDone:
			switch msg.String() {
			case "enter", "esc", "q":
				return s, pop
			}
		}
	}
	var cmd tea.Cmd
	s.view, cmd = s.view.Update(msg)
	return s, cmd
}

func (s *planScreen) resultText(msg appliedMsg) string {
	var b strings.Builder
	if msg.err != nil && msg.result.Failed == nil {
		b.WriteString(errorStyle.Render("✗ Nothing was applied: "+msg.err.Error()) + "\n")
		return b.String()
	}
	if s.prepared.Backup != nil {
		b.WriteString(okStyle.Render("✓ Backup written to "+s.prepared.Backup.Path) + "\n")
	}
	var summary bytes.Buffer
	msg.result.WriteSummary(&summary)
	if msg.result.Failed != nil {
		b.WriteString(errorStyle.Render("✗ " + summary.String()))
		b.WriteString(errorStyle.Render(msg.err.Error()) + "\n")
	} else {
		b.WriteString(okStyle.Render("✓ " + summary.String()))
	}
	if msg.result.AuditErr != nil {
		b.WriteString(warnStyle.Render("⚠ audit log not written: "+msg.result.AuditErr.Error()) + "\n")
	}

	if len(msg.creds) == 0 {
		return b.String()
	}
	b.WriteString("\n")
	switch {
	case len(s.targets) == 0:
		b.WriteString(formatCredentials(msg.creds))
	case msg.exportErr != nil:
		b.WriteString(formatReports(msg.reports))
		b.WriteString(errorStyle.Render("✗ Export failed: "+msg.exportErr.Error()) + "\n\n")
		b.WriteString(formatCredentials(msg.creds))
	default:
		b.WriteString(formatReports(msg.reports))
		b.WriteString(msg.exportOut)
	}
	return b.String()
}

func (s *planScreen) View() string {
	if s.state == planApplying {
		return s.env.loading("Applying… don't close launch-pg until this finishes.")
	}
	return s.view.View()
}

func (s *planScreen) Help() string {
	switch s.state {
	case planConfirm:
		return hint("y", "apply", "n/esc", "cancel", "↑/↓", "scroll")
	case planDone:
		return hint("↑/↓", "scroll", "enter/esc", "back")
	}
	return ""
}
