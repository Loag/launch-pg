package planner

import (
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/plan/actions"
	"launch-pg/internal/sqlgen"
)

// Clone copies a database with CREATE DATABASE … TEMPLATE.
type Clone struct {
	Source string
	Target string
	// TerminateConnections disconnects sessions on Source first; Postgres
	// refuses to copy a database that anyone is connected to.
	TerminateConnections bool
}

// CloneDatabase plans a copy owned by the source's owner, with the same
// explicit database-level grants.
func (p *Planner) CloneDatabase(c Clone, state model.ServerState) (Output, error) {
	if err := sqlgen.ValidateName("database", c.Target); err != nil {
		return Output{}, err
	}
	src, ok := state.Database(c.Source)
	if !ok {
		return Output{}, fmt.Errorf("database %s does not exist", c.Source)
	}
	if _, exists := state.Database(c.Target); exists {
		return Output{}, fmt.Errorf("database %s already exists", c.Target)
	}
	if src.Connections > 0 && !c.TerminateConnections {
		return Output{}, fmt.Errorf(
			"%s has %d active connection(s) and can't be copied while in use; "+
				"stop them or pass --terminate-connections", src.Name, src.Connections)
	}

	out := newOutput()
	// Cloning a non-template database requires owning it (via membership).
	out.Plan.Add(adminMembership(state.Info, src.Owner)...)
	if src.Connections > 0 {
		out.Plan.Add(actions.TerminateConnections{Name: src.Name})
	}
	out.Plan.Add(actions.CreateDatabase{Name: c.Target, Owner: src.Owner, Template: src.Name})
	addACLCopy(&out, src, c.Target)

	out.warn("database comments and per-database settings (e.g. search_path) are not copied")
	return out, nil
}

// addACLCopy reproduces the source's explicit database grants. New
// databases start with the default ACL, which lets PUBLIC connect.
func addACLCopy(out *Output, src model.DatabaseInfo, target string) {
	if src.DefaultACL {
		return
	}
	out.Plan.Add(actions.GrantDatabase{
		Name: target, Role: actions.Public, Privileges: []actions.Privilege{actions.PrivAll}, Revoke: true,
	})
	for _, grantee := range src.Grantees() {
		var privs []actions.Privilege
		for _, e := range src.ACL {
			if e.Grantee == grantee {
				privs = append(privs, actions.Privilege(e.Privilege))
			}
		}
		out.Plan.Add(actions.GrantDatabase{Name: target, Role: grantee, Privileges: privs})
	}
}
