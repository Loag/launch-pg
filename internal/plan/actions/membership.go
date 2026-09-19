package actions

import (
	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

// GrantMembership makes Member a member of Role.
//
// On Postgres 16+ a CREATEROLE admin only gets ADMIN OPTION on roles it
// creates; it needs SET to create objects owned by them (CREATE DATABASE …
// OWNER, ALTER DEFAULT PRIVILEGES FOR ROLE). ExplicitOptions emits the PG16
// "WITH INHERIT TRUE, SET TRUE" form; leave it false on 13–15.
type GrantMembership struct {
	Role            string
	Member          string
	ExplicitOptions bool
	Revoke          bool
}

func (a GrantMembership) Kind() string {
	if a.Revoke {
		return "revoke_membership"
	}
	return "grant_membership"
}

func (a GrantMembership) Database() string    { return "" }
func (a GrantMembership) Transactional() bool { return true }

func (a GrantMembership) Describe() string {
	if a.Revoke {
		return "remove " + a.Member + " from role " + a.Role
	}
	return "add " + a.Member + " to role " + a.Role
}

func (a GrantMembership) Validate() error {
	return required(map[string]string{"role": a.Role, "member": a.Member})
}

func (a GrantMembership) Statements() []plan.Statement {
	if a.Revoke {
		return []plan.Statement{plan.Stmt(
			"REVOKE " + sqlgen.Ident(a.Role) + " FROM " + sqlgen.Ident(a.Member))}
	}
	sql := "GRANT " + sqlgen.Ident(a.Role) + " TO " + sqlgen.Ident(a.Member)
	if a.ExplicitOptions {
		sql += " WITH INHERIT TRUE, SET TRUE"
	}
	return []plan.Statement{plan.Stmt(sql)}
}
