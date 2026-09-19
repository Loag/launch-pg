package planner

import (
	"reflect"
	"strings"
	"testing"

	"launch-pg/internal/model"
)

func withACL(state model.ServerState) model.ServerState {
	for i, d := range state.Databases {
		if d.Name == "myapp" {
			state.Databases[i].ACL = []model.ACLEntry{
				{Grantee: "myapp_owner", Privilege: "CONNECT"},
				{Grantee: "myapp_app", Privilege: "CONNECT"},
				{Grantee: "myapp_app", Privilege: "TEMPORARY"},
			}
		}
	}
	return state
}

func TestCloneCopiesOwnerAndGrants(t *testing.T) {
	out, err := newTestPlanner().CloneDatabase(Clone{Source: "myapp", Target: "myapp_copy"}, withACL(existingProject()))
	if err != nil {
		t.Fatal(err)
	}
	sql := allSQL(out.Plan)
	for _, want := range []string{
		`CREATE DATABASE "myapp_copy" OWNER "myapp_owner" TEMPLATE "myapp"`,
		`REVOKE ALL ON DATABASE "myapp_copy" FROM PUBLIC`,
		`GRANT CONNECT, TEMPORARY ON DATABASE "myapp_copy" TO "myapp_app"`,
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("missing %s\n%s", want, sql)
		}
	}
	if strings.Contains(sql, `TO "myapp_owner"`) {
		t.Error("owner's implicit privileges should not be granted explicitly")
	}
}

func TestCloneRefusesBusySourceUnlessTerminating(t *testing.T) {
	state := existingProject()
	for i := range state.Databases {
		if state.Databases[i].Name == "myapp" {
			state.Databases[i].Connections = 2
		}
	}
	pl := newTestPlanner()
	if _, err := pl.CloneDatabase(Clone{Source: "myapp", Target: "copy"}, state); err == nil {
		t.Error("expected error for busy source")
	}
	out, err := pl.CloneDatabase(Clone{Source: "myapp", Target: "copy", TerminateConnections: true}, state)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(allSQL(out.Plan), "pg_terminate_backend") {
		t.Error("expected terminate step")
	}
}

func TestTeardownWithRoles(t *testing.T) {
	out, err := newTestPlanner().DropDatabase(Teardown{Database: "myapp", WithRoles: true}, existingProject())
	if err != nil {
		t.Fatal(err)
	}
	got := kinds(out.Plan)
	want := []string{
		"grant_membership", "grant_membership", "grant_membership", // owner, app, old
		"drop_database",
		"drop_owned", "drop_owned", "drop_owned", // in postgres, per role
		"drop_role", "drop_role", "drop_role",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("kinds:\n got %v\nwant %v", got, want)
	}
	for _, a := range out.Plan.Actions {
		if a.Database() == "myapp" {
			t.Errorf("%s runs in the dropped database", a.Describe())
		}
	}
}

func TestTeardownGuards(t *testing.T) {
	pl := newTestPlanner()

	untagged := pg16Admin()
	untagged.Databases = append(untagged.Databases, model.DatabaseInfo{Name: "legacy", Owner: "bob"})
	if _, err := pl.DropDatabase(Teardown{Database: "legacy", WithRoles: true}, untagged); err == nil {
		t.Error("expected error: --with-roles on an untagged database")
	}

	busy := existingProject()
	for i := range busy.Databases {
		busy.Databases[i].Connections = 1
	}
	if _, err := pl.DropDatabase(Teardown{Database: "myapp"}, busy); err == nil {
		t.Error("expected error: open connections without --force")
	}

	tmpl := pg16Admin()
	tmpl.Databases = append(tmpl.Databases, model.DatabaseInfo{Name: "template1", IsTemplate: true})
	if _, err := pl.DropDatabase(Teardown{Database: "template1"}, tmpl); err == nil {
		t.Error("expected refusal to drop a template")
	}
}
