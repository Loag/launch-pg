package planner

import (
	"fmt"

	"launch-pg/internal/model"
)

// addPrune drops roles tagged for this project that are no longer desired,
// reassigning whatever they own to the project owner.
func (p *Planner) addPrune(out *Output, desired model.Project, owner model.RoleSpec, state model.ServerState) error {
	keep := make(map[string]bool, len(desired.Roles))
	for _, r := range desired.Roles {
		keep[r.Name] = true
	}

	for _, info := range state.Roles {
		m, tagged := info.Marker()
		if !tagged || m.Project != desired.Name || keep[info.Name] {
			continue
		}
		if _, exists := state.Role(owner.Name); !exists {
			return fmt.Errorf("cannot prune %s: owner role %s doesn't exist yet; apply without --prune first",
				info.Name, owner.Name)
		}
		drop, err := p.DropRole(RoleDrop{Name: info.Name, ReassignTo: owner.Name}, state)
		if err != nil {
			return fmt.Errorf("prune %s: %w", info.Name, err)
		}
		out.Plan.Add(drop.Plan.Actions...)
		out.Warnings = append(out.Warnings, drop.Warnings...)
	}
	return nil
}
