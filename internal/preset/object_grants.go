package preset

import (
	"launch-pg/internal/plan"
	"launch-pg/internal/plan/actions"
)

// objectGrants describes a schema-scoped preset: database privileges,
// schema privileges, and per-object-type privileges applied both to
// existing objects and, via default privileges, to future ones.
type objectGrants struct {
	database []actions.Privilege
	schema   []actions.Privilege
	objects  map[actions.ObjectType][]actions.Privilege
}

// objectOrder keeps generated SQL deterministic.
var objectOrder = []actions.ObjectType{actions.Tables, actions.Sequences, actions.Functions}

func (g objectGrants) build(t Target, role string, revoke bool) []plan.Action {
	out := []plan.Action{
		actions.GrantDatabase{Name: t.Database, Role: role, Privileges: g.database, Revoke: revoke},
		actions.GrantSchema{InDatabase: t.Database, Schema: t.Schema, Role: role, Privileges: g.schema, Revoke: revoke},
	}
	for _, obj := range objectOrder {
		privs, ok := g.objects[obj]
		if !ok {
			continue
		}
		out = append(out,
			actions.GrantAllInSchema{
				InDatabase: t.Database, Schema: t.Schema, Objects: obj,
				Role: role, Privileges: privs, Revoke: revoke,
			},
			actions.AlterDefaultPrivileges{
				InDatabase: t.Database, ForRole: t.Owner, Schema: t.Schema, Objects: obj,
				Role: role, Privileges: privs, Revoke: revoke,
			},
		)
	}
	return out
}
