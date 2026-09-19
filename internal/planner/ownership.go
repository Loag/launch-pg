package planner

import (
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/plan/actions"
)

// checkOwnership refuses to touch objects that belong to another project or
// that launch-pg must never manage, and warns when adopting untagged ones.
func checkOwnership(out *Output, desired model.Project, state model.ServerState) error {
	for _, r := range desired.Roles {
		info, exists := state.Role(r.Name)
		if !exists {
			continue
		}
		if _, err := manageableRole(r.Name, state); err != nil {
			return err
		}
		m, tagged := info.Marker()
		switch {
		case tagged && m.Project != desired.Name:
			return fmt.Errorf("role %s belongs to launch-pg project %q", r.Name, m.Project)
		case !tagged:
			out.warn(fmt.Sprintf("adopting existing role %s into project %s", r.Name, desired.Name))
		}
	}

	if d, exists := state.Database(desired.Database.Name); exists {
		m, tagged := d.Marker()
		switch {
		case tagged && m.Project != desired.Name:
			return fmt.Errorf("database %s belongs to launch-pg project %q", d.Name, m.Project)
		case !tagged:
			out.warn(fmt.Sprintf("adopting existing database %s into project %s", d.Name, desired.Name))
		}
	}
	return nil
}

// addMarkers tags roles and the database, skipping ones already correct.
func addMarkers(out *Output, desired model.Project, state model.ServerState) {
	for _, r := range desired.Roles {
		want := model.Marker{Project: desired.Name, Preset: r.Preset}.String()
		if info, ok := state.Role(r.Name); ok && info.Comment == want {
			continue
		}
		out.Plan.Add(actions.CommentOnRole{Name: r.Name, Comment: want})
	}

	want := model.Marker{Project: desired.Name}.String()
	if d, ok := state.Database(desired.Database.Name); ok && d.Comment == want {
		return
	}
	out.Plan.Add(actions.CommentOnDatabase{Name: desired.Database.Name, Comment: want})
}
