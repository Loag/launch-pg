package actions

import (
	"errors"
	"strings"

	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

// CreateDatabase creates a database. Template is optional; set it to clone.
type CreateDatabase struct {
	Name     string
	Owner    string
	Template string
	Encoding string
	Locale   string
}

func (a CreateDatabase) Kind() string        { return "create_database" }
func (a CreateDatabase) Database() string    { return "" }
func (a CreateDatabase) Transactional() bool { return false }

func (a CreateDatabase) Describe() string {
	if a.Template != "" {
		return "create database " + a.Name + " as a copy of " + a.Template
	}
	return "create database " + a.Name
}

func (a CreateDatabase) Validate() error {
	return required(map[string]string{"name": a.Name, "owner": a.Owner})
}

func (a CreateDatabase) Statements() []plan.Statement {
	parts := []string{"CREATE DATABASE", sqlgen.Ident(a.Name), "OWNER", sqlgen.Ident(a.Owner)}
	if a.Template != "" {
		parts = append(parts, "TEMPLATE", sqlgen.Ident(a.Template))
	}
	if a.Encoding != "" {
		parts = append(parts, "ENCODING", sqlgen.Literal(a.Encoding))
	}
	if a.Locale != "" {
		parts = append(parts, "LOCALE", sqlgen.Literal(a.Locale))
	}
	return []plan.Statement{plan.Stmt(strings.Join(parts, " "))}
}

// DropDatabase drops a database. Force terminates other connections (PG13+).
type DropDatabase struct {
	Name  string
	Force bool
}

func (a DropDatabase) Kind() string            { return "drop_database" }
func (a DropDatabase) Database() string        { return "" }
func (a DropDatabase) Transactional() bool     { return false }
func (a DropDatabase) Describe() string        { return "drop database " + a.Name }
func (a DropDatabase) DroppedDatabase() string { return a.Name }
func (a DropDatabase) Validate() error         { return required(map[string]string{"name": a.Name}) }

func (a DropDatabase) Statements() []plan.Statement {
	sql := "DROP DATABASE IF EXISTS " + sqlgen.Ident(a.Name)
	if a.Force {
		sql += " WITH (FORCE)"
	}
	return []plan.Statement{plan.Stmt(sql)}
}

// SetSearchPath sets the default search_path for sessions on a database.
type SetSearchPath struct {
	Name    string
	Schemas []string
}

func (a SetSearchPath) Kind() string        { return "set_search_path" }
func (a SetSearchPath) Database() string    { return "" }
func (a SetSearchPath) Transactional() bool { return true }

func (a SetSearchPath) Describe() string {
	return "set search_path of " + a.Name + " to " + strings.Join(a.Schemas, ", ")
}

func (a SetSearchPath) Validate() error {
	if len(a.Schemas) == 0 {
		return errors.New("at least one schema is required")
	}
	return required(map[string]string{"name": a.Name})
}

func (a SetSearchPath) Statements() []plan.Statement {
	quoted := make([]string, len(a.Schemas))
	for i, s := range a.Schemas {
		quoted[i] = sqlgen.Ident(s)
	}
	return []plan.Statement{plan.Stmt(
		"ALTER DATABASE " + sqlgen.Ident(a.Name) + " SET search_path TO " + strings.Join(quoted, ", "))}
}

// TerminateConnections disconnects every other session on a database,
// e.g. before using it as a clone template.
type TerminateConnections struct {
	Name string
}

func (a TerminateConnections) Kind() string        { return "terminate_connections" }
func (a TerminateConnections) Database() string    { return "" }
func (a TerminateConnections) Transactional() bool { return true }
func (a TerminateConnections) Describe() string    { return "terminate connections to " + a.Name }
func (a TerminateConnections) Validate() error     { return required(map[string]string{"name": a.Name}) }

func (a TerminateConnections) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = " +
			sqlgen.Literal(a.Name) + " AND pid <> pg_backend_pid()")}
}
