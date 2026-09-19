// Package sqlgen builds SQL text safely. Every identifier or literal that
// reaches generated SQL must pass through this package.
package sqlgen

import (
	"strings"

	"github.com/jackc/pgx/v5"
)

// Ident quotes a single identifier, e.g. my"db → "my""db".
func Ident(name string) string {
	return pgx.Identifier{name}.Sanitize()
}

// QualifiedIdent quotes a dotted name such as schema.table.
func QualifiedIdent(parts ...string) string {
	return pgx.Identifier(parts).Sanitize()
}

// Literal quotes a string literal. Assumes standard_conforming_strings=on
// (the default since 9.1); if the value contains a backslash, an E-prefixed
// escape-string literal is emitted so the result is correct regardless of that setting.
func Literal(value string) string {
	escaped := strings.ReplaceAll(value, "'", "''")
	if strings.Contains(escaped, `\`) {
		return "E'" + strings.ReplaceAll(escaped, `\`, `\\`) + "'"
	}
	return "'" + escaped + "'"
}
