package preset

import (
	"launch-pg/internal/model"
	"launch-pg/internal/plan"
	"launch-pg/internal/plan/actions"
)

// Owner gives full control by membership in the database owner role. The
// owner role itself needs nothing: it owns the database and schema.
type Owner struct{}

func (Owner) Name() string { return model.PresetOwner }

func (Owner) Description() string {
	return "owns the database and schema; use for migrations"
}

func (Owner) Grant(t Target, role string) []plan.Action {
	if role == t.Owner {
		return nil
	}
	return []plan.Action{actions.GrantMembership{Role: t.Owner, Member: role}}
}

func (Owner) Revoke(t Target, role string) []plan.Action {
	if role == t.Owner {
		return nil
	}
	return []plan.Action{actions.GrantMembership{Role: t.Owner, Member: role, Revoke: true}}
}
