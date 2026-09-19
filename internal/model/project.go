package model

import (
	"fmt"

	"launch-pg/internal/sqlgen"
)

// Preset names.
const (
	PresetOwner     = "owner"
	PresetReadWrite = "readwrite"
	PresetReadOnly  = "readonly"
)

// Project is the desired state for one app: a database and its roles.
type Project struct {
	Name     string
	Database DatabaseSpec
	Roles    []RoleSpec
}

type DatabaseSpec struct {
	Name       string
	Schema     string
	Encoding   string
	Locale     string
	Extensions []string
}

type RoleSpec struct {
	Name            string
	Preset          string
	Login           bool
	ConnectionLimit int
}

// Owner returns the single role with the owner preset.
func (p Project) Owner() (RoleSpec, error) {
	var owners []RoleSpec
	for _, r := range p.Roles {
		if r.Preset == PresetOwner {
			owners = append(owners, r)
		}
	}
	if len(owners) != 1 {
		return RoleSpec{}, fmt.Errorf("project %s must have exactly one owner role, has %d", p.Name, len(owners))
	}
	return owners[0], nil
}

// Validate checks names and structure.
func (p Project) Validate() error {
	if err := sqlgen.ValidateName("database", p.Database.Name); err != nil {
		return err
	}
	if err := sqlgen.ValidateName("schema", p.Database.Schema); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, r := range p.Roles {
		if err := sqlgen.ValidateName("role", r.Name); err != nil {
			return err
		}
		if seen[r.Name] {
			return fmt.Errorf("role %s is listed twice", r.Name)
		}
		seen[r.Name] = true
	}
	for _, ext := range p.Database.Extensions {
		if ext == "" {
			return fmt.Errorf("empty extension name")
		}
	}
	_, err := p.Owner()
	return err
}
