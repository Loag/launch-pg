package planner

import (
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/plan/actions"
	"launch-pg/internal/preset"
)

// addRoleDrift corrects existing roles that differ from the desired state.
// A changed preset is detected from the role's marker; the old preset is
// revoked before the new one is granted (the caller adds grants after this).
func (p *Planner) addRoleDrift(out *Output, desired model.Project, state model.ServerState, target preset.Target) {
	for _, r := range desired.Roles {
		info, exists := state.Role(r.Name)
		if !exists {
			continue
		}

		if m, ok := info.Marker(); ok && m.Preset != "" && m.Preset != r.Preset {
			old, err := p.presets.Get(m.Preset)
			if err != nil {
				out.warn(fmt.Sprintf("role %s was tagged with unknown preset %q; not revoking it", r.Name, m.Preset))
			} else {
				out.Plan.Add(old.Revoke(target, r.Name)...)
			}
		}

		if want := unlimitedAsMinusOne(r.ConnectionLimit); info.ConnectionLimit != want {
			out.Plan.Add(actions.SetConnectionLimit{Name: r.Name, Limit: want})
		}

		if info.CanLogin != r.Login {
			out.warn(fmt.Sprintf("role %s login=%t but spec says login=%t; launch-pg does not change this",
				r.Name, info.CanLogin, r.Login))
		}
	}
}

// unlimitedAsMinusOne maps the spec's 0 ("not set") to Postgres' -1.
func unlimitedAsMinusOne(limit int) int {
	if limit <= 0 {
		return -1
	}
	return limit
}
