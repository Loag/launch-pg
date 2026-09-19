// Package tui is an interactive terminal front end over the same services
// as the CLI. It contains no provisioning logic of its own: every change
// is a service.Prepared plan, shown and confirmed before it is applied.
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/app"
)

// Run starts the TUI and blocks until the user quits.
func Run(ctx context.Context, services app.Services) error {
	e := &env{ctx: ctx, svc: services}
	root := newAppModel(e, newServersScreen(e))
	_, err := tea.NewProgram(root, tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	return err
}
