package actions

import (
	"fmt"
	"strings"

	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

const redactedPassword = "PASSWORD '********'"

// CreateRole creates a role. Password should be a SCRAM verifier so the
// plaintext never reaches the server; it is redacted in printed output.
type CreateRole struct {
	Name            string
	Login           bool
	Password        string
	ConnectionLimit int // 0 is treated as unlimited (-1)
}

func (a CreateRole) Kind() string        { return "create_role" }
func (a CreateRole) Database() string    { return "" }
func (a CreateRole) Transactional() bool { return true }

func (a CreateRole) Describe() string {
	kind := "group role"
	if a.Login {
		kind = "login role"
	}
	return fmt.Sprintf("create %s %s", kind, a.Name)
}

func (a CreateRole) Validate() error {
	if err := required(map[string]string{"name": a.Name}); err != nil {
		return err
	}
	if a.Password != "" && !a.Login {
		return fmt.Errorf("role %s: password set on a NOLOGIN role", a.Name)
	}
	return nil
}

func (a CreateRole) Statements() []plan.Statement {
	opts := []string{"NOLOGIN"}
	if a.Login {
		opts = []string{"LOGIN"}
	}
	opts = append(opts, "NOSUPERUSER", "NOCREATEDB", "NOCREATEROLE", "INHERIT")
	if a.ConnectionLimit > 0 {
		opts = append(opts, fmt.Sprintf("CONNECTION LIMIT %d", a.ConnectionLimit))
	}

	base := "CREATE ROLE " + sqlgen.Ident(a.Name) + " WITH " + strings.Join(opts, " ")
	if a.Password == "" {
		return []plan.Statement{plan.Stmt(base)}
	}
	return []plan.Statement{plan.Redacted(
		base+" PASSWORD "+sqlgen.Literal(a.Password),
		base+" "+redactedPassword,
	)}
}

// SetRolePassword changes a role's password (rotation).
type SetRolePassword struct {
	Name     string
	Password string
}

func (a SetRolePassword) Kind() string        { return "set_role_password" }
func (a SetRolePassword) Database() string    { return "" }
func (a SetRolePassword) Transactional() bool { return true }
func (a SetRolePassword) Describe() string    { return "set password for " + a.Name }

func (a SetRolePassword) Validate() error {
	return required(map[string]string{"name": a.Name, "password": a.Password})
}

func (a SetRolePassword) Statements() []plan.Statement {
	base := "ALTER ROLE " + sqlgen.Ident(a.Name) + " WITH"
	return []plan.Statement{plan.Redacted(
		base+" PASSWORD "+sqlgen.Literal(a.Password),
		base+" "+redactedPassword,
	)}
}

// SetConnectionLimit changes a role's connection limit; -1 is unlimited.
type SetConnectionLimit struct {
	Name  string
	Limit int
}

func (a SetConnectionLimit) Kind() string        { return "set_connection_limit" }
func (a SetConnectionLimit) Database() string    { return "" }
func (a SetConnectionLimit) Transactional() bool { return true }

func (a SetConnectionLimit) Describe() string {
	return fmt.Sprintf("set connection limit of %s to %d", a.Name, a.Limit)
}

func (a SetConnectionLimit) Validate() error {
	if a.Limit < -1 {
		return fmt.Errorf("connection limit %d is invalid", a.Limit)
	}
	return required(map[string]string{"name": a.Name})
}

func (a SetConnectionLimit) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt(fmt.Sprintf(
		"ALTER ROLE %s WITH CONNECTION LIMIT %d", sqlgen.Ident(a.Name), a.Limit))}
}

// DropRole drops a role. Objects it owns must be reassigned or dropped first
// (see ReassignOwned / DropOwned), in every database.
type DropRole struct {
	Name string
}

func (a DropRole) Kind() string        { return "drop_role" }
func (a DropRole) Database() string    { return "" }
func (a DropRole) Transactional() bool { return true }
func (a DropRole) Describe() string    { return "drop role " + a.Name }
func (a DropRole) Validate() error     { return required(map[string]string{"name": a.Name}) }

func (a DropRole) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt("DROP ROLE IF EXISTS " + sqlgen.Ident(a.Name))}
}
