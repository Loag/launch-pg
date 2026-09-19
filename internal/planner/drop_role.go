package planner

import (
	"errors"
	"fmt"
	"strings"

	"launch-pg/internal/model"
	"launch-pg/internal/plan/actions"
)

// RoleDrop describes how to remove a role. Exactly what happens to objects
// the role owns must be chosen explicitly.
type RoleDrop struct {
	Name        string
	ReassignTo  string // move owned objects to this role…
	DropObjects bool   // …or drop them
}

// DropRole plans REASSIGN OWNED (optional) and DROP OWNED in every database
// the admin can reach, then DROP ROLE.
func (p *Planner) DropRole(d RoleDrop, state model.ServerState) (Output, error) {
	if _, err := manageableRole(d.Name, state); err != nil {
		return Output{}, err
	}
	if (d.ReassignTo == "") == !d.DropObjects {
		return Output{}, errors.New("choose exactly one of reassigning owned objects or dropping them")
	}
	if d.ReassignTo != "" {
		if _, ok := state.Role(d.ReassignTo); !ok {
			return Output{}, fmt.Errorf("role %s does not exist", d.ReassignTo)
		}
	}

	var ownedDBs []string
	for _, db := range state.Databases {
		if db.Owner == d.Name {
			ownedDBs = append(ownedDBs, db.Name)
		}
	}
	if len(ownedDBs) > 0 && d.ReassignTo == "" {
		return Output{}, fmt.Errorf("role %s owns database(s) %s; drop them first or reassign to another role",
			d.Name, strings.Join(ownedDBs, ", "))
	}

	out := newOutput()
	members := []string{d.Name}
	if d.ReassignTo != "" {
		members = append(members, d.ReassignTo)
	}
	out.Plan.Add(adminMembership(state.Info, members...)...)

	for _, db := range state.Databases {
		if db.IsTemplate || !db.AllowConn {
			continue
		}
		if !db.CanConnect {
			out.warn(fmt.Sprintf("no CONNECT on database %s; objects or grants of %s there are not cleaned up "+
				"and DROP ROLE may fail", db.Name, d.Name))
			continue
		}
		if d.ReassignTo != "" {
			out.Plan.Add(actions.ReassignOwned{InDatabase: db.Name, Role: d.Name, To: d.ReassignTo})
		}
		out.Plan.Add(actions.DropOwned{InDatabase: db.Name, Role: d.Name})
	}
	out.Plan.Add(actions.DropRole{Name: d.Name})
	return out, nil
}
