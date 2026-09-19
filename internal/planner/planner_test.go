package planner

import (
	"reflect"
	"strings"
	"testing"

	"launch-pg/internal/model"
	"launch-pg/internal/password"
	"launch-pg/internal/plan"
	"launch-pg/internal/preset"
)

type fixedIssuer struct{}

func (fixedIssuer) Issue() (password.Secret, error) {
	return password.Secret{Plain: "plain", Verifier: "SCRAM-SHA-256$verifier"}, nil
}

func newTestPlanner() *Planner { return New(preset.Default(), fixedIssuer{}) }

func pg16Admin() model.ServerState {
	return model.ServerState{
		Info:      model.ServerInfo{VersionNum: 160002, Version: "16.2", CurrentUser: "admin", CreateDB: true, CreateRole: true},
		Databases: []model.DatabaseInfo{{Name: "postgres", Owner: "postgres", AllowConn: true, CanConnect: true}},
		Roles:     []model.RoleInfo{{Name: "admin", CanLogin: true, CreateDB: true, CreateRole: true}},
	}
}

func testProject() model.Project {
	return model.Project{
		Name: "myapp",
		Database: model.DatabaseSpec{
			Name: "myapp", Schema: "public", Extensions: []string{"pgcrypto"},
		},
		Roles: []model.RoleSpec{
			{Name: "myapp_app", Preset: model.PresetReadWrite, Login: true},
			{Name: "myapp_owner", Preset: model.PresetOwner, Login: true},
		},
	}
}

func kinds(p *plan.Plan) []string {
	out := make([]string, len(p.Actions))
	for i, a := range p.Actions {
		out[i] = a.Kind()
	}
	return out
}

func allSQL(p *plan.Plan) string {
	var b strings.Builder
	for _, a := range p.Actions {
		for _, s := range a.Statements() {
			b.WriteString(s.SQL + "\n")
		}
	}
	return b.String()
}

func TestProjectOnPG16AsNonSuperuser(t *testing.T) {
	out, err := newTestPlanner().Project(testProject(), pg16Admin(), ProjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := out.Plan.Validate(); err != nil {
		t.Fatal(err)
	}

	got := kinds(out.Plan)
	wantPrefix := []string{
		"create_role", "create_role", "grant_membership", "create_database",
		"revoke_database", "create_extension",
	}
	if !reflect.DeepEqual(got[:len(wantPrefix)], wantPrefix) {
		t.Errorf("action order = %v", got)
	}
	if first := out.Plan.Actions[0].Describe(); !strings.Contains(first, "myapp_owner") {
		t.Errorf("owner must be created first, got %q", first)
	}

	sql := allSQL(out.Plan)
	for _, want := range []string{
		`GRANT "myapp_owner" TO "admin" WITH INHERIT TRUE, SET TRUE`,
		`CREATE DATABASE "myapp" OWNER "myapp_owner"`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE "myapp_owner" IN SCHEMA "public" GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "myapp_app"`,
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("plan SQL missing:\n%s\n--- got ---\n%s", want, sql)
		}
	}
	if len(out.Secrets) != 2 || out.Secrets["myapp_app"] != "plain" {
		t.Errorf("secrets = %v", out.Secrets)
	}
	if len(out.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", out.Warnings)
	}
}

func TestProjectIsAdditive(t *testing.T) {
	state := pg16Admin()
	state.Roles = append(state.Roles,
		model.RoleInfo{Name: "myapp_owner", CanLogin: true, ConnectionLimit: -1,
			Comment: "launch-pg:v1 preset=owner project=myapp"},
		model.RoleInfo{Name: "myapp_app", CanLogin: true, ConnectionLimit: -1,
			Comment: "launch-pg:v1 preset=readwrite project=myapp"})
	state.Databases = append(state.Databases, model.DatabaseInfo{
		Name: "myapp", Owner: "myapp_owner", Comment: "launch-pg:v1 project=myapp"})

	out, err := newTestPlanner().Project(testProject(), state, ProjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range kinds(out.Plan) {
		if k == "create_role" || k == "create_database" {
			t.Errorf("re-planning an existing project should not create %s", k)
		}
	}
	if len(out.Secrets) != 0 {
		t.Errorf("no new passwords expected, got %v", out.Secrets)
	}
}

func TestPublicSchemaWarningOnPG14NonSuperuser(t *testing.T) {
	state := pg16Admin()
	state.Info.VersionNum, state.Info.Version = 140010, "14.10"

	out, err := newTestPlanner().Project(testProject(), state, ProjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Warnings) != 1 {
		t.Errorf("warnings = %v", out.Warnings)
	}
	if strings.Contains(allSQL(out.Plan), "WITH INHERIT TRUE") {
		t.Error("PG16 membership syntax used on PG14")
	}
}

func TestNonPublicSchemaSetsSearchPath(t *testing.T) {
	p := testProject()
	p.Database.Schema = "app"
	out, err := newTestPlanner().Project(p, pg16Admin(), ProjectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	sql := allSQL(out.Plan)
	for _, want := range []string{
		`CREATE SCHEMA IF NOT EXISTS "app" AUTHORIZATION "myapp_owner"`,
		`ALTER DATABASE "myapp" SET search_path TO "app", "public"`,
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %s", want)
		}
	}
}

func TestDropRoleRequiresExplicitObjectHandling(t *testing.T) {
	state := pg16Admin()
	state.Roles = append(state.Roles, model.RoleInfo{Name: "old", CanLogin: true})
	pl := newTestPlanner()

	if _, err := pl.DropRole(RoleDrop{Name: "old"}, state); err == nil {
		t.Error("expected error without reassign/drop choice")
	}
	if _, err := pl.DropRole(RoleDrop{Name: "admin", DropObjects: true}, state); err == nil {
		t.Error("expected refusal to drop the admin role")
	}

	out, err := pl.DropRole(RoleDrop{Name: "old", DropObjects: true}, state)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"grant_membership", "drop_owned", "drop_role"}
	if got := kinds(out.Plan); !reflect.DeepEqual(got, want) {
		t.Errorf("kinds = %v, want %v", got, want)
	}
}
