package tui

import (
	"bytes"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/export"
	"launch-pg/internal/model"
	"launch-pg/internal/plan"
	"launch-pg/internal/service"
)

// prepare runs a planning call off the UI goroutine.
func prepare(fn func() (service.Prepared, error), targets []export.Target) tea.Cmd {
	return func() tea.Msg {
		p, err := fn()
		if err != nil {
			return errMsg{err}
		}
		return preparedMsg{prepared: p, targets: targets}
	}
}

// appliedMsg carries everything the plan screen shows after applying.
type appliedMsg struct {
	result    plan.Result
	err       error
	creds     []model.Credential
	reports   []export.Report
	exportOut string
	exportErr error
}

// applyCmd applies a plan and then exports any new credentials. Rendered
// exports without out= are captured and shown instead of written to the
// terminal, which the TUI owns.
func applyCmd(e *env, p service.Prepared, targets []export.Target) tea.Cmd {
	return func() tea.Msg {
		res, err := e.svc.Applier.Apply(e.ctx, p)
		msg := appliedMsg{result: res, err: err, creds: p.Credentials(res.Applied)}
		if len(targets) > 0 && len(msg.creds) > 0 {
			var buf bytes.Buffer
			msg.reports, msg.exportErr = e.svc.Exporter.Export(e.ctx, &buf, msg.creds, targets)
			msg.exportOut = buf.String()
		}
		return msg
	}
}

// parseTargets parses export target specs, skipping empty ones.
func parseTargets(e *env, specs ...string) ([]export.Target, error) {
	var nonEmpty []string
	for _, s := range specs {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	return e.svc.Exporter.Parse(nonEmpty)
}

// exportCmd exports roles to a target: rotating first if the target needs
// a password, otherwise exporting directly.
func exportCmd(e *env, server string, roles []string, database, target string) tea.Cmd {
	targets, err := e.svc.Exporter.Parse([]string{target})
	if err != nil {
		return func() tea.Msg { return errMsg{err} }
	}
	if e.svc.Exporter.NeedsPassword(targets) {
		return prepare(func() (service.Prepared, error) {
			return e.svc.Roles.PrepareRotate(e.ctx, server, roles, database)
		}, targets)
	}
	return func() tea.Msg {
		creds, err := e.svc.Roles.Credentials(e.ctx, server, roles, database)
		if err != nil {
			return errMsg{err}
		}
		var buf bytes.Buffer
		reports, err := e.svc.Exporter.Export(e.ctx, &buf, creds, targets)
		if err != nil {
			return errMsg{err}
		}
		return showMsg{title: "Export", body: formatReports(reports) + "\n" + buf.String()}
	}
}

func formatReports(reports []export.Report) string {
	var b strings.Builder
	for _, r := range reports {
		fmt.Fprintf(&b, "Exported %s → %s (%s)\n", r.Role, r.Location, r.Target)
	}
	return b.String()
}

func formatCredentials(creds []model.Credential) string {
	var b strings.Builder
	b.WriteString("Credentials (shown once; launch-pg does not store them, rotate if lost):\n")
	for _, c := range creds {
		fmt.Fprintf(&b, "\n  role:     %s\n  password: %s\n  url:      %s\n", c.User, c.Password, c.URL())
	}
	return b.String()
}

// isYes reads a y/n form field.
func isYes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes", "true", "1":
		return true
	}
	return false
}
