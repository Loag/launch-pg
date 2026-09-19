// Package planner compares desired state with a server snapshot and builds
// the plan that closes the gap. It performs no I/O.
package planner

import (
	"launch-pg/internal/model"
	"launch-pg/internal/password"
	"launch-pg/internal/plan"
	"launch-pg/internal/plan/actions"
	"launch-pg/internal/preset"
)

// SecretIssuer creates new role passwords.
type SecretIssuer interface {
	Issue() (password.Secret, error)
}

// Planner builds plans. It is stateless apart from its collaborators.
type Planner struct {
	presets *preset.Registry
	secrets SecretIssuer
}

func New(presets *preset.Registry, secrets SecretIssuer) *Planner {
	return &Planner{presets: presets, secrets: secrets}
}

// Output is a plan plus what the caller needs to report after applying it.
type Output struct {
	Plan *plan.Plan
	// Secrets maps role name → plaintext password for roles whose
	// password this plan sets. Shown to the user once, never stored.
	Secrets  map[string]string
	Warnings []string
}

func newOutput() Output {
	return Output{Plan: &plan.Plan{}, Secrets: map[string]string{}}
}

func (o *Output) warn(msg string) {
	o.Warnings = append(o.Warnings, msg)
}

// adminMembership lets a non-superuser admin act as role: required to
// create objects owned by it, alter its default privileges, and reassign or
// drop what it owns. Always emitted: on PG16+ an existing membership may
// lack SET/INHERIT, and re-granting is harmless on older versions.
func adminMembership(info model.ServerInfo, roles ...string) []plan.Action {
	if info.Superuser {
		return nil
	}
	out := make([]plan.Action, 0, len(roles))
	for _, role := range roles {
		if role == info.CurrentUser {
			continue // a role can't be a member of itself, nor needs to be
		}
		out = append(out, actions.GrantMembership{
			Role:            role,
			Member:          info.CurrentUser,
			ExplicitOptions: info.Major() >= 16,
		})
	}
	return out
}
