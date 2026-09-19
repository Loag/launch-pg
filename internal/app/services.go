// Package app defines the use-case surface the TUI depends on. The
// composition root supplies the implementations.
package app

import (
	"context"
	"io"

	"launch-pg/internal/config"
	"launch-pg/internal/export"
	"launch-pg/internal/model"
	"launch-pg/internal/plan"
	"launch-pg/internal/planner"
	"launch-pg/internal/service"
	"launch-pg/internal/templates"
)

// ServerManager manages registered servers.
type ServerManager interface {
	Add(server config.Server, password string, makeDefault bool) error
	List() ([]config.Server, string, error)
	Remove(name string) error
	SetDefault(name string) error
	Test(ctx context.Context, name string) (model.ServerInfo, error)
}

// ProjectManager provisions projects.
type ProjectManager interface {
	PrepareNew(ctx context.Context, req service.NewProjectRequest) (service.Prepared, error)
	PrepareApply(ctx context.Context, req service.ApplyRequest) (service.Prepared, error)
}

// RoleManager manages individual roles.
type RoleManager interface {
	List(ctx context.Context, server string) ([]model.RoleInfo, error)
	PrepareCreate(ctx context.Context, server string, spec model.RoleSpec, credentialDB string) (service.Prepared, error)
	PrepareRotate(ctx context.Context, server string, roles []string, credentialDB string) (service.Prepared, error)
	ProjectLoginRoles(ctx context.Context, server, project string) ([]string, string, error)
	Credentials(ctx context.Context, server string, roles []string, database string) ([]model.Credential, error)
	PrepareLimit(ctx context.Context, server, role string, limit int) (service.Prepared, error)
	PreparePreset(ctx context.Context, server string, change planner.PresetChange) (service.Prepared, error)
	PrepareDrop(ctx context.Context, server string, drop planner.RoleDrop) (service.Prepared, error)
}

// DatabaseManager lists, clones and drops databases.
type DatabaseManager interface {
	List(ctx context.Context, server string) ([]model.DatabaseInfo, error)
	PrepareClone(ctx context.Context, server string, c planner.Clone) (service.Prepared, error)
	PrepareDrop(ctx context.Context, req service.DropRequest) (service.Prepared, error)
}

// CredentialExporter sends credentials to files and secret stores.
type CredentialExporter interface {
	Parse(specs []string) ([]export.Target, error)
	NeedsPassword(targets []export.Target) bool
	Export(ctx context.Context, stdout io.Writer, creds []model.Credential, targets []export.Target) ([]export.Report, error)
	Kinds() []string
}

// PlanApplier applies prepared plans.
type PlanApplier interface {
	Apply(ctx context.Context, p service.Prepared) (plan.Result, error)
}

// PresetInfo describes a preset for help output and pickers.
type PresetInfo struct {
	Name        string
	Description string
}

// Services is everything a front end needs.
type Services struct {
	Servers   ServerManager
	Projects  ProjectManager
	Roles     RoleManager
	Databases DatabaseManager
	Applier   PlanApplier
	Exporter  CredentialExporter
	Templates templates.Source
	Presets   []PresetInfo
	Version   string
}
