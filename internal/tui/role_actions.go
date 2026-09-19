package tui

import (
	"errors"
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"launch-pg/internal/export"
	"launch-pg/internal/model"
	"launch-pg/internal/planner"
	"launch-pg/internal/service"
)

// roleActions builds role operations for one server. Shared by the roles
// screen and the database detail screen, which keep databases and roles
// up to date so forms can offer them as choices.
type roleActions struct {
	env       *env
	server    string
	databases []string
	roles     []string
}

// forRole returns every action that applies to one role.
func (a roleActions) forRole(role model.RoleInfo, database string) []action {
	name := role.Name
	actions := []action{
		{key: "g", label: "Grant a preset…", desc: "Give " + name + " owner, read-write or read-only access to a database.",
			open: func() screen { return a.presetForm(name, database, false) }},
		{key: "v", label: "Revoke a preset…", desc: "Remove a preset's privileges from " + name + ".",
			open: func() screen { return a.presetForm(name, database, true) }},
	}
	if role.CanLogin {
		actions = append(actions,
			action{key: "o", label: "Rotate password", desc: "Generate a new password; the old one stops working.",
				run: func() tea.Cmd { return a.rotate(name, database) }},
			action{key: "e", label: "Export credentials…", desc: "Send credentials to Kubernetes, .env, Vault, AWS… (rotates unless External Secrets).",
				open: func() screen { return a.exportMenu([]string{name}, database, "export "+name) }},
		)
	}
	return append(actions,
		action{key: "l", label: "Set connection limit…", desc: "Cap how many sessions " + name + " may open.",
			open: func() screen { return a.limitForm(role) }},
		action{key: "x", label: "Drop role…", desc: "Delete " + name + "; choose what happens to what it owns.",
			open: func() screen { return a.dropForm(name) }},
	)
}

func (a roleActions) create(database string) action {
	return action{key: "n", label: "Create a role…", desc: "A standalone role with a generated password.",
		open: func() screen { return a.createForm(database) }}
}

func (a roleActions) databaseOptions(none string) []option {
	opts := []option{}
	if none != "" {
		opts = append(opts, option{"", none, ""})
	}
	for _, d := range a.databases {
		opts = append(opts, option{d, d, ""})
	}
	return opts
}

func (a roleActions) createForm(database string) screen {
	return newForm(a.env, "create role",
		"The role starts with no privileges; grant it a preset afterwards.",
		"Plan changes",
		[]field{
			textInput("Role name", "", "reporting", "Lowercase letters, digits and _."),
			toggle("Can log in", true, "Turn off for a group role that only holds privileges."),
			textInput("Connection limit", "0", "", "0 means unlimited."),
			choice("Database for the connection URL", a.databaseOptions("(none)"), database, "Only affects the credentials shown or exported."),
			toggle("Export the new credentials", false, "Next you'll choose where."),
		},
		func(v []string) tea.Cmd {
			limit, err := strconv.Atoi(v[2])
			if err != nil {
				return errCmd(errors.New("connection limit must be a number"))
			}
			spec := model.RoleSpec{Name: v[0], Login: isYes(v[1]), ConnectionLimit: limit}
			return withOptionalExport(a.env, isYes(v[4]), "export "+v[0], v[3], func(targets []export.Target) tea.Cmd {
				return prepare(func() (service.Prepared, error) {
					return a.env.svc.Roles.PrepareCreate(a.env.ctx, a.server, spec, v[3])
				}, targets)
			})
		})
}

func (a roleActions) dropForm(role string) screen {
	opts := []option{{"", "Nobody — drop them", "DROP OWNED: deletes tables and other objects the role owns, in every database."}}
	for _, r := range a.roles {
		if r != role {
			opts = append(opts, option{r, "Reassign to " + r, "Ownership moves to " + r + "; nothing is deleted."})
		}
	}
	return newForm(a.env, "drop role "+role,
		"Removes "+role+" from the server. Its grants are revoked in every database.",
		"Plan changes",
		[]field{
			choice("What happens to objects "+role+" owns?", opts, "", ""),
			toggle("I understand owned objects may be deleted", false, "Required when not reassigning."),
		},
		func(v []string) tea.Cmd {
			drop := planner.RoleDrop{Name: role, ReassignTo: v[0]}
			if v[0] == "" {
				if !isYes(v[1]) {
					return errCmd(errors.New("confirm that owned objects may be deleted, or pick a role to reassign to"))
				}
				drop.DropObjects = true
			}
			return prepare(func() (service.Prepared, error) {
				return a.env.svc.Roles.PrepareDrop(a.env.ctx, a.server, drop)
			}, nil)
		})
}

func (a roleActions) presetForm(role, database string, revoke bool) screen {
	verb, intro := "grant", "Presets also cover tables created later by the database owner."
	if revoke {
		verb, intro = "revoke", "Removes exactly the privileges the preset grants."
	}
	presets := make([]option, len(a.env.svc.Presets))
	for i, p := range a.env.svc.Presets {
		presets[i] = option{p.Name, p.Name, p.Description}
	}
	return newForm(a.env, fmt.Sprintf("%s preset: %s", verb, role), intro, "Plan changes",
		[]field{
			choice("Preset", presets, "readwrite", ""),
			choice("Database", a.databaseOptions(""), database, ""),
			textInput("Schema", "public", "", ""),
		},
		func(v []string) tea.Cmd {
			return prepare(func() (service.Prepared, error) {
				return a.env.svc.Roles.PreparePreset(a.env.ctx, a.server, planner.PresetChange{
					Role: role, Preset: v[0], Database: v[1], Schema: v[2], Revoke: revoke,
				})
			}, nil)
		})
}

func (a roleActions) rotate(role, database string) tea.Cmd {
	return prepare(func() (service.Prepared, error) {
		return a.env.svc.Roles.PrepareRotate(a.env.ctx, a.server, []string{role}, database)
	}, nil)
}

func (a roleActions) exportMenu(roles []string, database, title string) screen {
	return exportMenu(a.env, title, database, func(target string) tea.Cmd {
		return exportCmd(a.env, a.server, roles, database, target)
	})
}

func (a roleActions) limitForm(role model.RoleInfo) screen {
	current := "-1"
	if role.ConnectionLimit >= 0 {
		current = strconv.Itoa(role.ConnectionLimit)
	}
	return newForm(a.env, "connection limit: "+role.Name, "", "Plan changes",
		[]field{textInput("Max connections", current, "", "-1 means unlimited.")},
		func(v []string) tea.Cmd {
			limit, err := strconv.Atoi(v[0])
			if err != nil {
				return errCmd(fmt.Errorf("not a number: %q", v[0]))
			}
			return prepare(func() (service.Prepared, error) {
				return a.env.svc.Roles.PrepareLimit(a.env.ctx, a.server, role.Name, limit)
			}, nil)
		})
}

func errCmd(err error) tea.Cmd {
	return func() tea.Msg { return errMsg{err} }
}
