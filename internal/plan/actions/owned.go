package actions

import (
	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

// ReassignOwned moves ownership of everything Role owns in InDatabase to To.
type ReassignOwned struct {
	InDatabase string
	Role       string
	To         string
}

func (a ReassignOwned) Kind() string        { return "reassign_owned" }
func (a ReassignOwned) Database() string    { return a.InDatabase }
func (a ReassignOwned) Transactional() bool { return true }

func (a ReassignOwned) Describe() string {
	return "reassign objects owned by " + a.Role + " to " + a.To + " in " + a.InDatabase
}

func (a ReassignOwned) Validate() error {
	return required(map[string]string{"database": a.InDatabase, "role": a.Role, "to": a.To})
}

func (a ReassignOwned) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt(
		"REASSIGN OWNED BY " + sqlgen.Ident(a.Role) + " TO " + sqlgen.Ident(a.To))}
}

// DropOwned drops objects owned by Role in InDatabase and revokes its
// privileges there, which DROP ROLE requires.
type DropOwned struct {
	InDatabase string
	Role       string
}

func (a DropOwned) Kind() string        { return "drop_owned" }
func (a DropOwned) Database() string    { return a.InDatabase }
func (a DropOwned) Transactional() bool { return true }

func (a DropOwned) Describe() string {
	return "drop objects and privileges of " + a.Role + " in " + a.InDatabase
}

func (a DropOwned) Validate() error {
	return required(map[string]string{"database": a.InDatabase, "role": a.Role})
}

func (a DropOwned) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt("DROP OWNED BY " + sqlgen.Ident(a.Role))}
}
