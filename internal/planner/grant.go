package planner

import (
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/preset"
)

// PresetChange grants or revokes a preset for a role on a database schema.
type PresetChange struct {
	Role     string
	Preset   string
	Database string
	Schema   string
	Revoke   bool
}

// ChangePreset plans a preset grant or revoke. Default privileges are tied
// to the database owner, since that role creates the objects.
func (p *Planner) ChangePreset(c PresetChange, state model.ServerState) (Output, error) {
	if _, err := manageableRole(c.Role, state); err != nil {
		return Output{}, err
	}
	db, ok := state.Database(c.Database)
	if !ok {
		return Output{}, fmt.Errorf("database %s does not exist", c.Database)
	}
	pr, err := p.presets.Get(c.Preset)
	if err != nil {
		return Output{}, err
	}

	target := preset.Target{Database: db.Name, Schema: c.Schema, Owner: db.Owner}
	out := newOutput()
	out.Plan.Add(adminMembership(state.Info, db.Owner)...)
	if c.Revoke {
		out.Plan.Add(pr.Revoke(target, c.Role)...)
	} else {
		out.Plan.Add(pr.Grant(target, c.Role)...)
	}
	return out, nil
}
