package actions

import (
	"launch-pg/internal/plan"
	"launch-pg/internal/sqlgen"
)

// CommentOnRole sets a role's comment (used for launch-pg ownership markers).
type CommentOnRole struct {
	Name    string
	Comment string
}

func (a CommentOnRole) Kind() string        { return "comment_on_role" }
func (a CommentOnRole) Database() string    { return "" }
func (a CommentOnRole) Transactional() bool { return true }
func (a CommentOnRole) Describe() string    { return "tag role " + a.Name + " (" + a.Comment + ")" }
func (a CommentOnRole) Validate() error     { return required(map[string]string{"name": a.Name}) }

func (a CommentOnRole) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt(
		"COMMENT ON ROLE " + sqlgen.Ident(a.Name) + " IS " + sqlgen.Literal(a.Comment))}
}

// CommentOnDatabase sets a database's comment.
type CommentOnDatabase struct {
	Name    string
	Comment string
}

func (a CommentOnDatabase) Kind() string        { return "comment_on_database" }
func (a CommentOnDatabase) Database() string    { return "" }
func (a CommentOnDatabase) Transactional() bool { return true }
func (a CommentOnDatabase) Describe() string {
	return "tag database " + a.Name + " (" + a.Comment + ")"
}
func (a CommentOnDatabase) Validate() error { return required(map[string]string{"name": a.Name}) }

func (a CommentOnDatabase) Statements() []plan.Statement {
	return []plan.Statement{plan.Stmt(
		"COMMENT ON DATABASE " + sqlgen.Ident(a.Name) + " IS " + sqlgen.Literal(a.Comment))}
}
