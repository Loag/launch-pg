package actions

import (
	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

// CreateExtension installs an extension into a database. Schema is optional.
type CreateExtension struct {
	InDatabase string
	Name       string
	Schema     string
}

func (a CreateExtension) Kind() string        { return "create_extension" }
func (a CreateExtension) Database() string    { return a.InDatabase }
func (a CreateExtension) Transactional() bool { return true }
func (a CreateExtension) Describe() string {
	return "install extension " + a.Name + " in " + a.InDatabase
}

func (a CreateExtension) Validate() error {
	return required(map[string]string{"database": a.InDatabase, "name": a.Name})
}

func (a CreateExtension) Statements() []plan.Statement {
	sql := "CREATE EXTENSION IF NOT EXISTS " + sqlgen.Ident(a.Name)
	if a.Schema != "" {
		sql += " SCHEMA " + sqlgen.Ident(a.Schema)
	}
	return []plan.Statement{plan.Stmt(sql)}
}
