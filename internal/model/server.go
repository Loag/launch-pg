// Package model holds plain domain types shared across packages.
package model

import "fmt"

// MinServerVersion is the oldest supported Postgres (13.0).
const MinServerVersion = 130000

// ServerInfo describes the connected server and the admin role's powers.
type ServerInfo struct {
	VersionNum  int    // server_version_num, e.g. 160002
	Version     string // server_version, e.g. "16.2"
	CurrentUser string
	Superuser   bool
	CreateDB    bool
	CreateRole  bool
}

// Major returns the major version, e.g. 16.
func (s ServerInfo) Major() int { return s.VersionNum / 10000 }

// Problems lists reasons the tool cannot fully operate on this server.
func (s ServerInfo) Problems() []string {
	var problems []string
	if s.VersionNum < MinServerVersion {
		problems = append(problems, fmt.Sprintf("Postgres %s is unsupported (need 13+)", s.Version))
	}
	if !s.Superuser && !s.CreateDB {
		problems = append(problems, fmt.Sprintf("role %q lacks CREATEDB", s.CurrentUser))
	}
	if !s.Superuser && !s.CreateRole {
		problems = append(problems, fmt.Sprintf("role %q lacks CREATEROLE", s.CurrentUser))
	}
	return problems
}
