package preset

import (
	"launch-pg/internal/model"
	"launch-pg/internal/plan"
	"launch-pg/internal/plan/actions"
)

// ReadWrite can read and modify data but not change the schema.
type ReadWrite struct{}

var readWriteGrants = objectGrants{
	database: []actions.Privilege{actions.PrivConnect, actions.PrivTemporary},
	schema:   []actions.Privilege{actions.PrivUsage},
	objects: map[actions.ObjectType][]actions.Privilege{
		actions.Tables:    {actions.PrivSelect, actions.PrivInsert, actions.PrivUpdate, actions.PrivDelete},
		actions.Sequences: {actions.PrivUsage, actions.PrivSelect, actions.PrivUpdate},
		actions.Functions: {actions.PrivExecute},
	},
}

func (ReadWrite) Name() string { return model.PresetReadWrite }

func (ReadWrite) Description() string {
	return "read and write data (SELECT/INSERT/UPDATE/DELETE); no DDL"
}

func (ReadWrite) Grant(t Target, role string) []plan.Action {
	return readWriteGrants.build(t, role, false)
}

func (ReadWrite) Revoke(t Target, role string) []plan.Action {
	return readWriteGrants.build(t, role, true)
}
