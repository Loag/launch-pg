package service

import (
	"context"
	"fmt"

	"launch-pg/internal/config"
	"launch-pg/internal/model"
	"launch-pg/internal/planner"
	"launch-pg/internal/spec"
	"launch-pg/internal/templates"
)

const fallbackTemplate = "webapp"

// ProjectService provisions whole projects.
type ProjectService struct {
	configs   config.Repository
	state     *StateLoader
	templates templates.Source
	specs     *spec.Resolver
	planner   *planner.Planner
}

func NewProjectService(
	configs config.Repository,
	state *StateLoader,
	templates templates.Source,
	specs *spec.Resolver,
	planner *planner.Planner,
) *ProjectService {
	return &ProjectService{configs: configs, state: state, templates: templates, specs: specs, planner: planner}
}

type ApplyRequest struct {
	// Server overrides the spec's server when set.
	Server string
	Spec   spec.File
	Prune  bool
}

// PrepareApply plans a declarative spec: create what is missing, correct
// drift, and optionally prune roles the spec no longer lists.
func (s *ProjectService) PrepareApply(ctx context.Context, req ApplyRequest) (Prepared, error) {
	project, err := s.specs.Resolve(req.Spec)
	if err != nil {
		return Prepared{}, err
	}
	serverName := req.Server
	if serverName == "" {
		serverName = req.Spec.Server
	}
	server, state, err := s.state.Load(ctx, serverName)
	if err != nil {
		return Prepared{}, err
	}

	out, err := s.planner.Project(project, state, planner.ProjectOptions{Prune: req.Prune})
	if err != nil {
		return Prepared{}, err
	}
	return newPrepared(server, out, project.Database.Name), nil
}

type NewProjectRequest struct {
	Server   string
	Name     string
	Template string
}

// PrepareNew plans a brand-new project. It refuses if the database or any
// of the roles already exist, so it never adopts something by accident.
func (s *ProjectService) PrepareNew(ctx context.Context, req NewProjectRequest) (Prepared, error) {
	tmpl, err := s.template(req.Template)
	if err != nil {
		return Prepared{}, err
	}
	project, err := tmpl.Expand(req.Name)
	if err != nil {
		return Prepared{}, err
	}

	server, state, err := s.state.Load(ctx, req.Server)
	if err != nil {
		return Prepared{}, err
	}
	if err := assertFresh(project, state); err != nil {
		return Prepared{}, err
	}

	out, err := s.planner.Project(project, state, planner.ProjectOptions{})
	if err != nil {
		return Prepared{}, err
	}
	return newPrepared(server, out, project.Database.Name), nil
}

func (s *ProjectService) template(name string) (templates.Template, error) {
	if name == "" {
		cfg, err := s.configs.Load()
		if err != nil {
			return templates.Template{}, err
		}
		name = cfg.DefaultTemplate
	}
	if name == "" {
		name = fallbackTemplate
	}
	return s.templates.Get(name)
}

func assertFresh(p model.Project, state model.ServerState) error {
	if _, exists := state.Database(p.Database.Name); exists {
		return fmt.Errorf("database %s already exists", p.Database.Name)
	}
	for _, r := range p.Roles {
		if _, exists := state.Role(r.Name); exists {
			return fmt.Errorf("role %s already exists", r.Name)
		}
	}
	return nil
}
