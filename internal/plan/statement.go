// Package plan models changes as an ordered list of Actions and applies them.
package plan

// Statement is one SQL statement. Display, when set, is a redacted form used
// for dry-run output and the audit log (e.g. passwords replaced).
type Statement struct {
	SQL     string
	Display string
}

// Stmt builds a Statement with nothing to redact.
func Stmt(sql string) Statement { return Statement{SQL: sql} }

// Redacted builds a Statement whose printed form differs from what runs.
func Redacted(sql, display string) Statement { return Statement{SQL: sql, Display: display} }

// String returns the safe-to-print form.
func (s Statement) String() string {
	if s.Display != "" {
		return s.Display
	}
	return s.SQL
}
