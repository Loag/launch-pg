package actions

import (
	"fmt"

	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

// GrantDatabase grants or revokes database-level privileges (CONNECT, …).
type GrantDatabase struct {
	Name       string
	Role       string
	Privileges []Privilege
	Revoke     bool
}

func (a GrantDatabase) Kind() string        { return grantKind("database", a.Revoke) }
func (a GrantDatabase) Database() string    { return "" }
func (a GrantDatabase) Transactional() bool { return true }

func (a GrantDatabase) Describe() string {
	return describeGrant(a.Revoke, a.Privileges, "database "+a.Name, a.Role)
}

func (a GrantDatabase) Validate() error {
	if err := required(map[string]string{"database": a.Name, "role": a.Role}); err != nil {
		return err
	}
	return validatePrivileges(databasePrivs, a.Privileges)
}

func (a GrantDatabase) Statements() []plan.Statement {
	verb, prep := grantVerb(a.Revoke)
	return []plan.Statement{plan.Stmt(fmt.Sprintf("%s %s ON DATABASE %s %s %s",
		verb, privilegeList(a.Privileges), sqlgen.Ident(a.Name), prep, grantee(a.Role)))}
}

// GrantSchema grants or revokes schema privileges (USAGE, CREATE).
type GrantSchema struct {
	InDatabase string
	Schema     string
	Role       string
	Privileges []Privilege
	Revoke     bool
}

func (a GrantSchema) Kind() string        { return grantKind("schema", a.Revoke) }
func (a GrantSchema) Database() string    { return a.InDatabase }
func (a GrantSchema) Transactional() bool { return true }

func (a GrantSchema) Describe() string {
	return describeGrant(a.Revoke, a.Privileges, "schema "+a.InDatabase+"."+a.Schema, a.Role)
}

func (a GrantSchema) Validate() error {
	if err := required(map[string]string{"database": a.InDatabase, "schema": a.Schema, "role": a.Role}); err != nil {
		return err
	}
	return validatePrivileges(schemaPrivs, a.Privileges)
}

func (a GrantSchema) Statements() []plan.Statement {
	verb, prep := grantVerb(a.Revoke)
	return []plan.Statement{plan.Stmt(fmt.Sprintf("%s %s ON SCHEMA %s %s %s",
		verb, privilegeList(a.Privileges), sqlgen.Ident(a.Schema), prep, grantee(a.Role)))}
}

// GrantAllInSchema grants or revokes privileges on every existing object of
// a type in a schema. Pair with AlterDefaultPrivileges for future objects.
type GrantAllInSchema struct {
	InDatabase string
	Schema     string
	Objects    ObjectType
	Role       string
	Privileges []Privilege
	Revoke     bool
}

func (a GrantAllInSchema) Kind() string        { return grantKind("all_in_schema", a.Revoke) }
func (a GrantAllInSchema) Database() string    { return a.InDatabase }
func (a GrantAllInSchema) Transactional() bool { return true }

func (a GrantAllInSchema) Describe() string {
	target := fmt.Sprintf("all %s in %s.%s", lower(a.Objects), a.InDatabase, a.Schema)
	return describeGrant(a.Revoke, a.Privileges, target, a.Role)
}

func (a GrantAllInSchema) Validate() error {
	if err := required(map[string]string{"database": a.InDatabase, "schema": a.Schema, "role": a.Role}); err != nil {
		return err
	}
	allowed, ok := objectPrivs[a.Objects]
	if !ok {
		return fmt.Errorf("unknown object type %q", a.Objects)
	}
	return validatePrivileges(allowed, a.Privileges)
}

func (a GrantAllInSchema) Statements() []plan.Statement {
	verb, prep := grantVerb(a.Revoke)
	return []plan.Statement{plan.Stmt(fmt.Sprintf("%s %s ON ALL %s IN SCHEMA %s %s %s",
		verb, privilegeList(a.Privileges), a.Objects, sqlgen.Ident(a.Schema), prep, grantee(a.Role)))}
}

func grantKind(target string, revoke bool) string {
	if revoke {
		return "revoke_" + target
	}
	return "grant_" + target
}

func describeGrant(revoke bool, privs []Privilege, target, role string) string {
	if revoke {
		return fmt.Sprintf("revoke %s on %s from %s", privilegeList(privs), target, role)
	}
	return fmt.Sprintf("grant %s on %s to %s", privilegeList(privs), target, role)
}
