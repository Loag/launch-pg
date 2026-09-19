package spec

import (
	"fmt"
	"slices"

	"launch-pg/internal/model"
	"launch-pg/internal/templates"
)

// Resolver turns a spec File into a validated model.Project, expanding its
// template (if any) and applying the file's overrides on top.
type Resolver struct {
	templates templates.Source
}

func NewResolver(templates templates.Source) *Resolver {
	return &Resolver{templates: templates}
}

func (r *Resolver) Resolve(f File) (model.Project, error) {
	project, err := r.base(f)
	if err != nil {
		return model.Project{}, err
	}
	applyDatabase(&project.Database, f.Database)
	for _, rs := range f.Roles {
		if err := applyRole(&project, rs); err != nil {
			return model.Project{}, err
		}
	}
	if err := project.Validate(); err != nil {
		return model.Project{}, fmt.Errorf("spec for %s: %w", f.Project, err)
	}
	return project, nil
}

// base expands the template, or starts empty when none is given.
func (r *Resolver) base(f File) (model.Project, error) {
	if f.Template == "" {
		return model.Project{
			Name:     f.Project,
			Database: model.DatabaseSpec{Name: f.Project, Schema: "public"},
		}, nil
	}
	tmpl, err := r.templates.Get(f.Template)
	if err != nil {
		return model.Project{}, err
	}
	// Validation happens after overrides, so a spec can complete a template
	// (e.g. one lacking an owner).
	return tmpl.Build(f.Project), nil
}

func applyDatabase(db *model.DatabaseSpec, s DatabaseSection) {
	if s.Name != "" {
		db.Name = s.Name
	}
	if s.Schema != "" {
		db.Schema = s.Schema
	}
	if s.Encoding != "" {
		db.Encoding = s.Encoding
	}
	if s.Locale != "" {
		db.Locale = s.Locale
	}
	for _, ext := range s.Extensions {
		if !slices.Contains(db.Extensions, ext) {
			db.Extensions = append(db.Extensions, ext)
		}
	}
}

func applyRole(p *model.Project, s RoleSection) error {
	if s.Name == "" {
		return fmt.Errorf("spec role without a name")
	}
	if s.Preset == "" {
		return fmt.Errorf("spec role %s: preset is required", s.Name)
	}
	role := model.RoleSpec{
		Name:            s.Name,
		Preset:          s.Preset,
		Login:           s.Login == nil || *s.Login,
		ConnectionLimit: s.ConnectionLimit,
	}
	for i, existing := range p.Roles {
		if existing.Name == s.Name {
			p.Roles[i] = role
			return nil
		}
	}
	p.Roles = append(p.Roles, role)
	return nil
}
