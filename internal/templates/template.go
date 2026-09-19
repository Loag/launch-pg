// Package templates loads project templates and expands them into projects.
package templates

import (
	"fmt"
	"strings"

	"launch-pg/internal/model"
)

const defaultSchema = "public"

// Template is a reusable project shape. Role names are <project>_<suffix>.
type Template struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Schema      string         `yaml:"schema"`
	Encoding    string         `yaml:"encoding"`
	Locale      string         `yaml:"locale"`
	Extensions  []string       `yaml:"extensions"`
	Roles       []RoleTemplate `yaml:"roles"`
}

type RoleTemplate struct {
	Suffix          string `yaml:"suffix"`
	Preset          string `yaml:"preset"`
	NoLogin         bool   `yaml:"noLogin"`
	ConnectionLimit int    `yaml:"connectionLimit"`
}

// Describe summarizes what the template creates, for display.
func (t Template) Describe() string {
	schema := t.Schema
	if schema == "" {
		schema = defaultSchema
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n\n", t.Name, t.Description)
	fmt.Fprintf(&b, "  schema:     %s\n", schema)
	if len(t.Extensions) > 0 {
		fmt.Fprintf(&b, "  extensions: %s\n", strings.Join(t.Extensions, ", "))
	}
	b.WriteString("  roles:\n")
	for _, r := range t.Roles {
		login := "login"
		if r.NoLogin {
			login = "no login"
		}
		limit := ""
		if r.ConnectionLimit > 0 {
			limit = fmt.Sprintf(", max %d connections", r.ConnectionLimit)
		}
		fmt.Fprintf(&b, "    <project>_%-8s %-10s %s%s\n", r.Suffix, r.Preset, login, limit)
	}
	return b.String()
}

// Expand builds the project for a given name and validates it.
func (t Template) Expand(project string) (model.Project, error) {
	p := t.Build(project)
	if err := p.Validate(); err != nil {
		return model.Project{}, fmt.Errorf("template %s: %w", t.Name, err)
	}
	return p, nil
}

// Build expands the template without validating, for callers that add to
// the project before validating it themselves.
func (t Template) Build(project string) model.Project {
	schema := t.Schema
	if schema == "" {
		schema = defaultSchema
	}
	p := model.Project{
		Name: project,
		Database: model.DatabaseSpec{
			Name:       project,
			Schema:     schema,
			Encoding:   t.Encoding,
			Locale:     t.Locale,
			Extensions: t.Extensions,
		},
	}
	for _, r := range t.Roles {
		p.Roles = append(p.Roles, model.RoleSpec{
			Name:            project + "_" + r.Suffix,
			Preset:          r.Preset,
			Login:           !r.NoLogin,
			ConnectionLimit: r.ConnectionLimit,
		})
	}
	return p
}
