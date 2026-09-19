package actions

import (
	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

// CreateSchema creates a schema owned by Owner inside InDatabase.
type CreateSchema struct {
	InDatabase string
	Name       string
	Owner      string
}

func (a CreateSchema) Kind() string        { return "create_schema" }
func (a CreateSchema) Database() string    { return a.InDatabase }
func (a CreateSchema) Transactional() bool { return true }
func (a CreateSchema) Describe() string    { return "create schema " + a.Name + " in " + a.InDatabase }

func (a CreateSchema) Validate() error {
	return required(map[string]string{"database": a.InDatabase, "name": a.Name, "owner": a.Owner})
}

func (a CreateSchema) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt(
		"CREATE SCHEMA IF NOT EXISTS " + sqlgen.Ident(a.Name) + " AUTHORIZATION " + sqlgen.Ident(a.Owner))}
}

// SetSchemaOwner changes a schema's owner, e.g. handing "public" to the
// project owner role.
type SetSchemaOwner struct {
	InDatabase string
	Name       string
	Owner      string
}

func (a SetSchemaOwner) Kind() string        { return "set_schema_owner" }
func (a SetSchemaOwner) Database() string    { return a.InDatabase }
func (a SetSchemaOwner) Transactional() bool { return true }

func (a SetSchemaOwner) Describe() string {
	return "make " + a.Owner + " owner of schema " + a.Name + " in " + a.InDatabase
}

func (a SetSchemaOwner) Validate() error {
	return required(map[string]string{"database": a.InDatabase, "name": a.Name, "owner": a.Owner})
}

func (a SetSchemaOwner) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt(
		"ALTER SCHEMA " + sqlgen.Ident(a.Name) + " OWNER TO " + sqlgen.Ident(a.Owner))}
}
