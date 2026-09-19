package planner

import (
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/plan/actions"
)

// Teardown drops a database and, optionally, its project's roles.
type Teardown struct {
	Database string
	// WithRoles also drops every role tagged with the database's project.
	WithRoles bool
	// Force terminates open connections (DROP DATABASE … WITH (FORCE)).
	Force bool
}

// DropDatabase plans a teardown. Project roles are cleaned out of every
// other reachable database (DROP OWNED) before being dropped.
func (p *Planner) DropDatabase(t Teardown, state model.ServerState) (Output, error) {
	db, ok := state.Database(t.Database)
	if !ok {
		return Output{}, fmt.Errorf("database %s does not exist", t.Database)
	}
	if db.IsTemplate {
		return Output{}, fmt.Errorf("refusing to drop template database %s", db.Name)
	}
	if db.Connections > 0 && !t.Force {
		return Output{}, fmt.Errorf("%s has %d active connection(s); stop them or pass --force", db.Name, db.Connections)
	}

	var roles []string
	if t.WithRoles {
		var err error
		if roles, err = projectRoles(db, state); err != nil {
			return Output{}, err
		}
	}

	out := newOutput()
	out.Plan.Add(adminMembership(state.Info, append([]string{db.Owner}, without(roles, db.Owner)...)...)...)
	out.Plan.Add(actions.DropDatabase{Name: db.Name, Force: t.Force})
	if len(roles) == 0 {
		return out, nil
	}

	for _, other := range state.Databases {
		if other.Name == db.Name || other.IsTemplate || !other.AllowConn {
			continue
		}
		if !other.CanConnect {
			out.warn(fmt.Sprintf("no CONNECT on database %s; grants held there by project roles are not "+
				"cleaned up and DROP ROLE may fail", other.Name))
			continue
		}
		for _, role := range roles {
			out.Plan.Add(actions.DropOwned{InDatabase: other.Name, Role: role})
		}
	}
	for _, role := range roles {
		out.Plan.Add(actions.DropRole{Name: role})
	}
	return out, nil
}

// projectRoles returns roles tagged with the database's project, refusing
// if any of them also owns another database.
func projectRoles(db model.DatabaseInfo, state model.ServerState) ([]string, error) {
	m, tagged := db.Marker()
	if !tagged || m.Project == "" {
		return nil, fmt.Errorf("database %s is not tagged by launch-pg, so its roles are unknown; "+
			"drop roles individually with `launch-pg role drop`", db.Name)
	}

	var roles []string
	for _, r := range state.Roles {
		if rm, ok := r.Marker(); !ok || rm.Project != m.Project {
			continue
		}
		if _, err := manageableRole(r.Name, state); err != nil {
			return nil, err
		}
		for _, other := range state.Databases {
			if other.Owner == r.Name && other.Name != db.Name {
				return nil, fmt.Errorf("role %s also owns database %s; drop or reassign it first", r.Name, other.Name)
			}
		}
		roles = append(roles, r.Name)
	}
	return roles, nil
}

func without(list []string, drop string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if s != drop {
			out = append(out, s)
		}
	}
	return out
}
