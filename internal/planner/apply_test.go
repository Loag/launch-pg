package planner

import (
	"slices"
	"strings"
	"testing"

	"launch-pg/internal/model"
)

// existingProject returns pg16Admin() with myapp already provisioned and tagged.
func existingProject() model.ServerState {
	state := pg16Admin()
	state.Roles = append(state.Roles,
		model.RoleInfo{Name: "myapp_owner", CanLogin: true, ConnectionLimit: -1,
			Comment: "launch-pg:v1 preset=owner project=myapp"},
		model.RoleInfo{Name: "myapp_app", CanLogin: true, ConnectionLimit: -1,
			Comment: "launch-pg:v1 preset=readwrite project=myapp"},
		model.RoleInfo{Name: "myapp_old", CanLogin: true, ConnectionLimit: -1,
			Comment: "launch-pg:v1 preset=readonly project=myapp"})
	state.Databases = append(state.Databases, model.DatabaseInfo{
		Name: "myapp", Owner: "myapp_owner", AllowConn: true, CanConnect: true,
		Comment: "launch-pg:v1 project=myapp"})
	return state
}

func TestReapplyUnchangedHasNoTagsOrWarnings(t *testing.T) {
	out, err := newTestPlanner().Project(testProject(), existingProject(), ProjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range kinds(out.Plan) {
		if strings.HasPrefix(k, "comment_on") || k == "drop_role" || k == "set_connection_limit" {
			t.Errorf("unexpected %s on unchanged re-apply", k)
		}
	}
	if len(out.Warnings) != 0 {
		t.Errorf("warnings = %v", out.Warnings)
	}
}

func TestPresetChangeRevokesOldPresetFirst(t *testing.T) {
	desired := testProject()
	desired.Roles[0].Preset = model.PresetReadOnly // myapp_app: readwrite → readonly

	out, err := newTestPlanner().Project(desired, existingProject(), ProjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	sql := allSQL(out.Plan)
	revoke := strings.Index(sql, `REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA "public" FROM "myapp_app"`)
	grant := strings.Index(sql, `GRANT SELECT ON ALL TABLES IN SCHEMA "public" TO "myapp_app"`)
	if revoke < 0 || grant < 0 || revoke > grant {
		t.Errorf("expected old preset revoked before new grant (revoke=%d grant=%d)\n%s", revoke, grant, sql)
	}
	if !strings.Contains(sql, `COMMENT ON ROLE "myapp_app" IS 'launch-pg:v1 preset=readonly project=myapp'`) {
		t.Error("marker not updated to new preset")
	}
}

func TestPruneDropsOnlyUndesiredProjectRoles(t *testing.T) {
	state := existingProject()
	state.Roles = append(state.Roles, model.RoleInfo{
		Name: "other_app", CanLogin: true, ConnectionLimit: -1,
		Comment: "launch-pg:v1 preset=readwrite project=other"})

	without, err := newTestPlanner().Project(testProject(), state, ProjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(kinds(without.Plan), "drop_role") {
		t.Error("roles dropped without --prune")
	}

	with, err := newTestPlanner().Project(testProject(), state, ProjectOptions{Prune: true})
	if err != nil {
		t.Fatal(err)
	}
	sql := allSQL(with.Plan)
	if !strings.Contains(sql, `REASSIGN OWNED BY "myapp_old" TO "myapp_owner"`) ||
		!strings.Contains(sql, `DROP ROLE IF EXISTS "myapp_old"`) {
		t.Errorf("myapp_old not pruned:\n%s", sql)
	}
	if strings.Contains(sql, `"other_app"`) {
		t.Error("pruned a role from another project")
	}
}

func TestRefusesRolesOfAnotherProject(t *testing.T) {
	state := pg16Admin()
	state.Roles = append(state.Roles, model.RoleInfo{
		Name: "myapp_app", CanLogin: true, Comment: "launch-pg:v1 project=other"})

	if _, err := newTestPlanner().Project(testProject(), state, ProjectOptions{}); err == nil {
		t.Error("expected error for role owned by another project")
	}
}

func TestMarkerRoundTrip(t *testing.T) {
	m := model.Marker{Project: "myapp", Preset: "readwrite"}
	got, ok := model.ParseMarker(m.String())
	if !ok || got != m {
		t.Errorf("round trip = %+v, %v", got, ok)
	}
	if _, ok := model.ParseMarker("some human comment"); ok {
		t.Error("foreign comment parsed as marker")
	}
}
