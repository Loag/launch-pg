package service

import (
	"context"
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/planner"
)

// RoleService manages individual roles.
type RoleService struct {
	state   *StateLoader
	planner *planner.Planner
}

func NewRoleService(state *StateLoader, planner *planner.Planner) *RoleService {
	return &RoleService{state: state, planner: planner}
}

func (s *RoleService) List(ctx context.Context, server string) ([]model.RoleInfo, error) {
	_, state, err := s.state.Load(ctx, server)
	if err != nil {
		return nil, err
	}
	return state.Roles, nil
}

// PrepareCreate plans a new role. credentialDB only affects the returned
// connection details.
func (s *RoleService) PrepareCreate(ctx context.Context, server string, spec model.RoleSpec, credentialDB string) (Prepared, error) {
	srv, state, err := s.state.Load(ctx, server)
	if err != nil {
		return Prepared{}, err
	}
	out, err := s.planner.CreateRole(spec, state)
	if err != nil {
		return Prepared{}, err
	}
	return newPrepared(srv, out, credentialDB), nil
}

func (s *RoleService) PrepareRotate(ctx context.Context, server string, roles []string, credentialDB string) (Prepared, error) {
	srv, state, err := s.state.Load(ctx, server)
	if err != nil {
		return Prepared{}, err
	}
	out, err := s.planner.RotatePasswords(roles, state)
	if err != nil {
		return Prepared{}, err
	}
	return newPrepared(srv, out, credentialDB), nil
}

// ProjectLoginRoles returns the login roles tagged with a project and the
// project's database.
func (s *RoleService) ProjectLoginRoles(ctx context.Context, server, project string) ([]string, string, error) {
	_, state, err := s.state.Load(ctx, server)
	if err != nil {
		return nil, "", err
	}
	var roles []string
	for _, r := range state.Roles {
		if m, ok := r.Marker(); ok && m.Project == project && r.CanLogin {
			roles = append(roles, r.Name)
		}
	}
	if len(roles) == 0 {
		return nil, "", fmt.Errorf("no login roles tagged with project %q", project)
	}
	database := ""
	for _, d := range state.Databases {
		if m, ok := d.Marker(); ok && m.Project == project {
			database = d.Name
		}
	}
	return roles, database, nil
}

// Credentials returns connection details without passwords, for export
// formats that only reference a secret stored elsewhere.
func (s *RoleService) Credentials(ctx context.Context, server string, roles []string, database string) ([]model.Credential, error) {
	srv, state, err := s.state.Load(ctx, server)
	if err != nil {
		return nil, err
	}
	creds := make([]model.Credential, 0, len(roles))
	for _, name := range roles {
		if _, ok := state.Role(name); !ok {
			return nil, fmt.Errorf("role %s does not exist", name)
		}
		creds = append(creds, model.Credential{
			Host: srv.Host, Port: srv.Port, Database: database, User: name, SSLMode: srv.SSLMode,
		})
	}
	return creds, nil
}

func (s *RoleService) PrepareLimit(ctx context.Context, server, role string, limit int) (Prepared, error) {
	srv, state, err := s.state.Load(ctx, server)
	if err != nil {
		return Prepared{}, err
	}
	out, err := s.planner.SetConnectionLimit(role, limit, state)
	if err != nil {
		return Prepared{}, err
	}
	return newPrepared(srv, out, ""), nil
}

func (s *RoleService) PreparePreset(ctx context.Context, server string, change planner.PresetChange) (Prepared, error) {
	srv, state, err := s.state.Load(ctx, server)
	if err != nil {
		return Prepared{}, err
	}
	out, err := s.planner.ChangePreset(change, state)
	if err != nil {
		return Prepared{}, err
	}
	return newPrepared(srv, out, change.Database), nil
}

func (s *RoleService) PrepareDrop(ctx context.Context, server string, drop planner.RoleDrop) (Prepared, error) {
	srv, state, err := s.state.Load(ctx, server)
	if err != nil {
		return Prepared{}, err
	}
	out, err := s.planner.DropRole(drop, state)
	if err != nil {
		return Prepared{}, err
	}
	return newPrepared(srv, out, ""), nil
}
