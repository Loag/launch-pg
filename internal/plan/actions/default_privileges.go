package actions

import (
	"fmt"
	"strings"

	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

// AlterDefaultPrivileges sets privileges on objects ForRole creates in the
// future, so grants survive new migrations. The admin must be a member of
// ForRole (see GrantMembership).
type AlterDefaultPrivileges struct {
	InDatabase string
	ForRole    string
	Schema     string
	Objects    ObjectType
	Role       string
	Privileges []Privilege
	Revoke     bool
}

func (a AlterDefaultPrivileges) Kind() string {
	if a.Revoke {
		return "revoke_default_privileges"
	}
	return "grant_default_privileges"
}

func (a AlterDefaultPrivileges) Database() string    { return a.InDatabase }
func (a AlterDefaultPrivileges) Transactional() bool { return true }

func (a AlterDefaultPrivileges) Describe() string {
	target := fmt.Sprintf("future %s created by %s in %s.%s", lower(a.Objects), a.ForRole, a.InDatabase, a.Schema)
	return describeGrant(a.Revoke, a.Privileges, target, a.Role)
}

func (a AlterDefaultPrivileges) Validate() error {
	err := required(map[string]string{
		"database": a.InDatabase, "forRole": a.ForRole, "schema": a.Schema, "role": a.Role,
	})
	if err != nil {
		return err
	}
	allowed, ok := objectPrivs[a.Objects]
	if !ok {
		return fmt.Errorf("unknown object type %q", a.Objects)
	}
	return validatePrivileges(allowed, a.Privileges)
}

func (a AlterDefaultPrivileges) Statements() []plan.Statement {
	verb, prep := grantVerb(a.Revoke)
	return []plan.Statement{plan.Stmt(fmt.Sprintf(
		"ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA %s %s %s ON %s %s %s",
		sqlgen.Ident(a.ForRole), sqlgen.Ident(a.Schema),
		verb, privilegeList(a.Privileges), a.Objects, prep, grantee(a.Role)))}
}

func lower(o ObjectType) string { return strings.ToLower(string(o)) }
