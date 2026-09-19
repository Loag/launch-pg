// Package actions holds the concrete plan.Action implementations.
package actions

import (
	"fmt"
	"strings"

	"launch-pg/internal/sqlgen"
)

// Privilege is a SQL privilege keyword. Keywords are emitted unquoted, so
// only values from the allowed sets below are accepted.
type Privilege string

const (
	PrivConnect   Privilege = "CONNECT"
	PrivTemporary Privilege = "TEMPORARY"
	PrivCreate    Privilege = "CREATE"
	PrivUsage     Privilege = "USAGE"
	PrivSelect    Privilege = "SELECT"
	PrivInsert    Privilege = "INSERT"
	PrivUpdate    Privilege = "UPDATE"
	PrivDelete    Privilege = "DELETE"
	PrivTruncate  Privilege = "TRUNCATE"
	PrivReference Privilege = "REFERENCES"
	PrivTrigger   Privilege = "TRIGGER"
	PrivExecute   Privilege = "EXECUTE"
	PrivAll       Privilege = "ALL"
)

// ObjectType is the kind of object a schema-wide grant targets.
type ObjectType string

const (
	Tables    ObjectType = "TABLES"
	Sequences ObjectType = "SEQUENCES"
	Functions ObjectType = "FUNCTIONS"
)

var (
	databasePrivs = privSet(PrivConnect, PrivTemporary, PrivCreate, PrivAll)
	schemaPrivs   = privSet(PrivUsage, PrivCreate, PrivAll)
	objectPrivs   = map[ObjectType]map[Privilege]bool{
		Tables: privSet(PrivSelect, PrivInsert, PrivUpdate, PrivDelete,
			PrivTruncate, PrivReference, PrivTrigger, PrivAll),
		Sequences: privSet(PrivUsage, PrivSelect, PrivUpdate, PrivAll),
		Functions: privSet(PrivExecute, PrivAll),
	}
)

func privSet(privs ...Privilege) map[Privilege]bool {
	m := make(map[Privilege]bool, len(privs))
	for _, p := range privs {
		m[p] = true
	}
	return m
}

func validatePrivileges(allowed map[Privilege]bool, privs []Privilege) error {
	if len(privs) == 0 {
		return fmt.Errorf("no privileges given")
	}
	for _, p := range privs {
		if !allowed[p] {
			return fmt.Errorf("privilege %q not allowed here", p)
		}
	}
	return nil
}

func privilegeList(privs []Privilege) string {
	parts := make([]string, len(privs))
	for i, p := range privs {
		parts[i] = string(p)
	}
	return strings.Join(parts, ", ")
}

// Public is the PUBLIC pseudo-role as a grantee.
const Public = "PUBLIC"

// grantee renders a role for GRANT/REVOKE; PUBLIC is a keyword, not an identifier.
func grantee(role string) string {
	if role == Public {
		return Public
	}
	return sqlgen.Ident(role)
}

// grantVerb returns the verb and preposition for a grant or revoke.
func grantVerb(revoke bool) (verb, prep string) {
	if revoke {
		return "REVOKE", "FROM"
	}
	return "GRANT", "TO"
}

func required(fields map[string]string) error {
	for name, v := range fields {
		if v == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}
