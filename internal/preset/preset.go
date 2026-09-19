// Package preset turns named permission levels into grant actions.
package preset

import (
	"fmt"
	"sort"

	"launch-pg/internal/plan"
)

// Target is where a preset applies: a schema in a database whose objects
// are owned (and will be created) by Owner.
type Target struct {
	Database string
	Schema   string
	Owner    string
}

// Preset is a named bundle of privileges.
type Preset interface {
	Name() string
	Description() string
	Grant(t Target, role string) []plan.Action
	Revoke(t Target, role string) []plan.Action
}

// Registry looks presets up by name.
type Registry struct {
	presets map[string]Preset
}

func NewRegistry(presets ...Preset) *Registry {
	m := make(map[string]Preset, len(presets))
	for _, p := range presets {
		m[p.Name()] = p
	}
	return &Registry{presets: m}
}

// Default returns the registry with the built-in presets.
func Default() *Registry {
	return NewRegistry(Owner{}, ReadWrite{}, ReadOnly{})
}

func (r *Registry) Get(name string) (Preset, error) {
	p, ok := r.presets[name]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q (have: %v)", name, r.Names())
	}
	return p, nil
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.presets))
	for n := range r.presets {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
