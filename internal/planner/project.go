package planner

import (
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/plan/actions"
	"launch-pg/internal/preset"
)

// ProjectOptions tune project planning.
type ProjectOptions struct {
	// Prune drops roles tagged for this project that the desired state no
	// longer lists. Their objects are reassigned to the owner first.
	Prune bool
}

// Project plans a project so the server matches desired:
//   - missing roles and the database are created
//   - grants, schema and extensions are (re)applied idempotently
//   - a role whose declared preset changed has the old preset revoked
//   - connection limits are corrected
//   - roles and database are tagged with launch-pg ownership markers
//   - with Prune, roles tagged for the project but not desired are dropped
func (p *Planner) Project(desired model.Project, state model.ServerState, opts ProjectOptions) (Output, error) {
	if err := desired.Validate(); err != nil {
		return Output{}, err
	}
	owner, _ := desired.Owner()
	db := desired.Database
	out := newOutput()

	if err := checkOwnership(&out, desired, state); err != nil {
		return Output{}, err
	}
	if err := p.addMissingRoles(&out, desired, owner, state); err != nil {
		return Output{}, err
	}
	out.Plan.Add(adminMembership(state.Info, owner.Name)...)
	addDatabase(&out, db, owner.Name, state)
	addSchema(&out, db, owner.Name, state.Info)
	for _, ext := range db.Extensions {
		out.Plan.Add(actions.CreateExtension{InDatabase: db.Name, Name: ext, Schema: db.Schema})
	}

	target := preset.Target{Database: db.Name, Schema: db.Schema, Owner: owner.Name}
	p.addRoleDrift(&out, desired, state, target)
	for _, r := range desired.Roles {
		pr, err := p.presets.Get(r.Preset)
		if err != nil {
			return Output{}, fmt.Errorf("role %s: %w", r.Name, err)
		}
		out.Plan.Add(pr.Grant(target, r.Name)...)
	}
	addMarkers(&out, desired, state)

	if opts.Prune {
		if err := p.addPrune(&out, desired, owner, state); err != nil {
			return Output{}, err
		}
	}
	return out, nil
}

// addDatabase creates the database if missing and locks out PUBLIC.
func addDatabase(out *Output, db model.DatabaseSpec, owner string, state model.ServerState) {
	if existing, ok := state.Database(db.Name); ok {
		if existing.Owner != owner {
			out.warn(fmt.Sprintf("database %s is owned by %s, not %s; ownership is left unchanged",
				db.Name, existing.Owner, owner))
		}
	} else {
		out.Plan.Add(actions.CreateDatabase{
			Name: db.Name, Owner: owner, Encoding: db.Encoding, Locale: db.Locale,
		})
	}
	out.Plan.Add(actions.GrantDatabase{
		Name: db.Name, Role: actions.Public, Privileges: []actions.Privilege{actions.PrivAll}, Revoke: true,
	})
}

// addMissingRoles creates roles that don't exist yet, owner first.
func (p *Planner) addMissingRoles(out *Output, desired model.Project, owner model.RoleSpec, state model.ServerState) error {
	ordered := []model.RoleSpec{owner}
	for _, r := range desired.Roles {
		if r.Name != owner.Name {
			ordered = append(ordered, r)
		}
	}
	for _, r := range ordered {
		if _, exists := state.Role(r.Name); exists {
			continue
		}
		create, err := p.createRole(out, r)
		if err != nil {
			return err
		}
		out.Plan.Add(create)
	}
	return nil
}

// createRole builds a CreateRole, issuing a password for login roles.
func (p *Planner) createRole(out *Output, r model.RoleSpec) (actions.CreateRole, error) {
	create := actions.CreateRole{Name: r.Name, Login: r.Login, ConnectionLimit: r.ConnectionLimit}
	if !r.Login {
		return create, nil
	}
	secret, err := p.secrets.Issue()
	if err != nil {
		return actions.CreateRole{}, fmt.Errorf("password for %s: %w", r.Name, err)
	}
	create.Password = secret.Verifier
	out.Secrets[r.Name] = secret.Plain
	return create, nil
}
