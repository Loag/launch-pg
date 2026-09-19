package actions

import (
	"strings"
	"testing"

	"launch-pg/internal/plan"
)

func sqlOf(a plan.Action) string {
	var parts []string
	for _, s := range a.Statements() {
		parts = append(parts, s.SQL)
	}
	return strings.Join(parts, "; ")
}

func TestStatements(t *testing.T) {
	cases := []struct {
		action plan.Action
		want   string
	}{
		{CreateRole{Name: "app", Login: true, ConnectionLimit: 5},
			`CREATE ROLE "app" WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT CONNECTION LIMIT 5`},
		{CreateDatabase{Name: "app", Owner: "app_owner", Encoding: "UTF8"},
			`CREATE DATABASE "app" OWNER "app_owner" ENCODING 'UTF8'`},
		{DropDatabase{Name: "app", Force: true},
			`DROP DATABASE IF EXISTS "app" WITH (FORCE)`},
		{GrantMembership{Role: "app_owner", Member: "admin", ExplicitOptions: true},
			`GRANT "app_owner" TO "admin" WITH INHERIT TRUE, SET TRUE`},
		{GrantDatabase{Name: "app", Role: Public, Privileges: []Privilege{PrivAll}, Revoke: true},
			`REVOKE ALL ON DATABASE "app" FROM PUBLIC`},
		{GrantAllInSchema{InDatabase: "app", Schema: "public", Objects: Tables, Role: "app_ro", Privileges: []Privilege{PrivSelect}},
			`GRANT SELECT ON ALL TABLES IN SCHEMA "public" TO "app_ro"`},
		{AlterDefaultPrivileges{InDatabase: "app", ForRole: "app_owner", Schema: "public", Objects: Sequences, Role: "app_rw", Privileges: []Privilege{PrivUsage, PrivSelect}},
			`ALTER DEFAULT PRIVILEGES FOR ROLE "app_owner" IN SCHEMA "public" GRANT USAGE, SELECT ON SEQUENCES TO "app_rw"`},
		{CreateExtension{InDatabase: "app", Name: "pgcrypto"},
			`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`},
	}
	for _, c := range cases {
		if err := c.action.Validate(); err != nil {
			t.Errorf("%s: unexpected validation error: %v", c.action.Kind(), err)
		}
		if got := sqlOf(c.action); got != c.want {
			t.Errorf("%s:\n got: %s\nwant: %s", c.action.Kind(), got, c.want)
		}
	}
}

func TestPasswordIsRedacted(t *testing.T) {
	a := CreateRole{Name: "app", Login: true, Password: "SCRAM-SHA-256$secret"}
	s := a.Statements()[0]
	if !strings.Contains(s.SQL, "SCRAM-SHA-256$secret") {
		t.Errorf("SQL missing password: %s", s.SQL)
	}
	if strings.Contains(s.String(), "secret") {
		t.Errorf("printed form leaks password: %s", s.String())
	}
}

func TestValidateRejectsBadPrivileges(t *testing.T) {
	bad := []plan.Action{
		GrantDatabase{Name: "app", Role: "r", Privileges: []Privilege{PrivSelect}},
		GrantSchema{InDatabase: "app", Schema: "s", Role: "r", Privileges: []Privilege{"DROP TABLE x;--"}},
		GrantAllInSchema{InDatabase: "app", Schema: "s", Objects: Functions, Role: "r", Privileges: []Privilege{PrivSelect}},
		CreateRole{Name: "r", Password: "x"},
	}
	for _, a := range bad {
		if err := a.Validate(); err == nil {
			t.Errorf("%s: expected validation error", a.Kind())
		}
	}
}
