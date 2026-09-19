package preset

import (
	"launch-pg/internal/model"
	"launch-pg/internal/plan"
	"launch-pg/internal/plan/actions"
)

// ReadOnly can only read data.
type ReadOnly struct{}

var readOnlyGrants = objectGrants{
	database: []actions.Privilege{actions.PrivConnect},
	schema:   []actions.Privilege{actions.PrivUsage},
	objects: map[actions.ObjectType][]actions.Privilege{
		actions.Tables:    {actions.PrivSelect},
		actions.Sequences: {actions.PrivSelect},
	},
}

func (ReadOnly) Name() string        { return model.PresetReadOnly }
func (ReadOnly) Description() string { return "read data only (SELECT)" }

func (ReadOnly) Grant(t Target, role string) []plan.Action {
	return readOnlyGrants.build(t, role, false)
}

func (ReadOnly) Revoke(t Target, role string) []plan.Action {
	return readOnlyGrants.build(t, role, true)
}
