package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/export"
	"launch-pg/internal/model"
	"launch-pg/internal/planner"
	"launch-pg/internal/service"
	"launch-pg/internal/spec"
)

// withOptionalExport runs plan directly, or first walks the export builder
// when the user asked to export, then plans with that target.
func withOptionalExport(e *env, wantExport bool, title, database string,
	plan func(targets []export.Target) tea.Cmd) tea.Cmd {
	if !wantExport {
		return plan(nil)
	}
	return replace(exportMenu(e, title, database, func(target string) tea.Cmd {
		targets, err := parseTargets(e, target)
		if err != nil {
			return errCmd(err)
		}
		return plan(targets)
	}))
}

func templateOptions(e *env) []option {
	list, err := e.svc.Templates.List()
	if err != nil {
		return []option{{"webapp", "webapp", "Owner, read-write app role, read-only role."}}
	}
	opts := make([]option, len(list))
	for i, t := range list {
		opts[i] = option{t.Name, t.Name, t.Description}
	}
	return opts
}

func newProjectForm(e *env, server string) screen {
	return newForm(e, "new project",
		"Creates a database and its roles from a template. Nothing changes until you confirm the plan.",
		"Plan changes",
		[]field{
			textInput("Project name", "", "myapp", "Used as the database name and role prefix (myapp_owner, myapp_app…). Lowercase letters, digits, _."),
			choice("Template", templateOptions(e), "webapp", ""),
			toggle("Export the new credentials", false, "Next you'll choose where: Kubernetes, .env, Vault, AWS…"),
		},
		func(v []string) tea.Cmd {
			name := v[0]
			return withOptionalExport(e, isYes(v[2]), "export "+name, name, func(targets []export.Target) tea.Cmd {
				return prepare(func() (service.Prepared, error) {
					return e.svc.Projects.PrepareNew(e.ctx, service.NewProjectRequest{
						Server: server, Name: name, Template: v[1],
					})
				}, targets)
			})
		})
}

// applySpecForm plans a declarative project spec against this server.
func applySpecForm(e *env, server string) screen {
	return newForm(e, "apply spec",
		"Makes the server match a project.yaml. Safe to re-run; you'll see the plan first.",
		"Plan changes",
		[]field{
			textInput("Spec file", "", "./project.yaml", "~/ is expanded."),
			toggle("Drop project roles the spec no longer lists", false, "Their objects are reassigned to the project owner first."),
			toggle("Also export new credentials somewhere else", false, "The spec's own export: list is always used."),
		},
		func(v []string) tea.Cmd {
			f, err := spec.Load(expandHome(v[0]))
			if err != nil {
				return errCmd(err)
			}
			if f.Server != "" && f.Server != server {
				return errCmd(fmt.Errorf("spec targets server %q but you are on %q", f.Server, server))
			}
			return withOptionalExport(e, isYes(v[2]), "export "+f.Project, f.Project, func(extra []export.Target) tea.Cmd {
				targets, err := parseTargets(e, f.Export...)
				if err != nil {
					return errCmd(err)
				}
				return prepare(func() (service.Prepared, error) {
					return e.svc.Projects.PrepareApply(e.ctx, service.ApplyRequest{
						Server: server, Spec: f, Prune: isYes(v[1]),
					})
				}, append(targets, extra...))
			})
		})
}

func cloneForm(e *env, server string, src model.DatabaseInfo) screen {
	return newForm(e, "clone "+src.Name,
		"Copies the database with the same owner and access. Postgres can't copy a database that has open connections.",
		"Plan changes",
		[]field{
			textInput("New database name", src.Name+"_copy", "", ""),
			toggle("Disconnect sessions on "+src.Name+" first", false,
				fmt.Sprintf("It has %d open connection(s).", src.Connections)),
		},
		func(v []string) tea.Cmd {
			return prepare(func() (service.Prepared, error) {
				return e.svc.Databases.PrepareClone(e.ctx, server, planner.Clone{
					Source: src.Name, Target: v[0], TerminateConnections: isYes(v[1]),
				})
			}, nil)
		})
}

func dropDatabaseForm(e *env, server string, db model.DatabaseInfo) screen {
	withRolesHint := "This database isn't tagged by launch-pg, so its roles can't be found automatically."
	if p := db.Project(); p != "" {
		withRolesHint = fmt.Sprintf("Drops every role tagged with project %q.", p)
	}
	return newForm(e, "drop "+db.Name,
		"Permanently deletes "+db.Name+". You'll see the plan before anything happens.",
		"Plan changes",
		[]field{
			toggle("Back up with pg_dump first", true, "Restore later with pg_restore. If the backup fails, nothing is dropped."),
			textInput("Backup file", "", "default: ~/.local/state/launch-pg/backups/…", ""),
			toggle("Also drop the project's roles", false, withRolesHint),
			toggle("Disconnect open sessions", false, fmt.Sprintf("It has %d open connection(s).", db.Connections)),
		},
		func(v []string) tea.Cmd {
			return prepare(func() (service.Prepared, error) {
				return e.svc.Databases.PrepareDrop(e.ctx, service.DropRequest{
					Server: server,
					Teardown: planner.Teardown{
						Database: db.Name, WithRoles: isYes(v[2]), Force: isYes(v[3]),
					},
					Backup:     isYes(v[0]),
					BackupPath: expandHome(v[1]),
				})
			}, nil)
		})
}

// expandHome turns a leading ~/ into the home directory.
func expandHome(path string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return path
}
