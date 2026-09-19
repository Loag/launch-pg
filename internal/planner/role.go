package planner

import (
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/plan/actions"
	"launch-pg/internal/sqlgen"
)

// CreateRole plans a standalone role.
func (p *Planner) CreateRole(spec model.RoleSpec, state model.ServerState) (Output, error) {
	if err := sqlgen.ValidateName("role", spec.Name); err != nil {
		return Output{}, err
	}
	if _, exists := state.Role(spec.Name); exists {
		return Output{}, fmt.Errorf("role %s already exists", spec.Name)
	}
	out := newOutput()
	create, err := p.createRole(&out, spec)
	if err != nil {
		return Output{}, err
	}
	out.Plan.Add(create)
	return out, nil
}

// RotatePasswords plans new passwords for existing login roles.
func (p *Planner) RotatePasswords(names []string, state model.ServerState) (Output, error) {
	if len(names) == 0 {
		return Output{}, fmt.Errorf("no roles to rotate")
	}
	out := newOutput()
	for _, name := range names {
		role, err := manageableRole(name, state)
		if err != nil {
			return Output{}, err
		}
		if !role.CanLogin {
			return Output{}, fmt.Errorf("role %s cannot log in; it has no password to rotate", name)
		}
		secret, err := p.secrets.Issue()
		if err != nil {
			return Output{}, err
		}
		out.Plan.Add(actions.SetRolePassword{Name: name, Password: secret.Verifier})
		out.Secrets[name] = secret.Plain
	}
	return out, nil
}

// SetConnectionLimit plans a new connection limit; -1 means unlimited.
func (p *Planner) SetConnectionLimit(name string, limit int, state model.ServerState) (Output, error) {
	if _, err := manageableRole(name, state); err != nil {
		return Output{}, err
	}
	out := newOutput()
	out.Plan.Add(actions.SetConnectionLimit{Name: name, Limit: limit})
	return out, nil
}

// manageableRole returns an existing role launch-pg is allowed to change:
// never a superuser and never the admin role itself.
func manageableRole(name string, state model.ServerState) (model.RoleInfo, error) {
	role, ok := state.Role(name)
	if !ok {
		return model.RoleInfo{}, fmt.Errorf("role %s does not exist", name)
	}
	if role.Superuser {
		return model.RoleInfo{}, fmt.Errorf("refusing to manage superuser role %s", name)
	}
	if name == state.Info.CurrentUser {
		return model.RoleInfo{}, fmt.Errorf("refusing to manage the admin role %s itself", name)
	}
	return role, nil
}
