package service

import (
	"launch-pg/internal/config"
	"launch-pg/internal/model"
	"launch-pg/internal/plan"
	"launch-pg/internal/plan/actions"
	"launch-pg/internal/planner"
)

// Prepared is a plan ready to show, confirm and apply.
type Prepared struct {
	Server   config.Server
	Plan     *plan.Plan
	Warnings []string
	// Backup, when set, is taken after confirmation and before the plan
	// runs; if it fails nothing is applied.
	Backup *BackupStep
	// secrets maps role → plaintext for roles whose password the plan sets.
	secrets map[string]string
	// credentialDB is the database named in returned credentials.
	credentialDB string
}

// BackupStep is a pg_dump of one database to Path.
type BackupStep struct {
	Database string
	Path     string
}

func newPrepared(server config.Server, out planner.Output, credentialDB string) Prepared {
	return Prepared{
		Server:       server,
		Plan:         out.Plan,
		Warnings:     out.Warnings,
		secrets:      out.Secrets,
		credentialDB: credentialDB,
	}
}

// Credentials returns connection details for every role whose password
// was actually set by the applied actions.
func (p Prepared) Credentials(applied []plan.Action) []model.Credential {
	var creds []model.Credential
	for _, a := range applied {
		var role string
		switch act := a.(type) {
		case actions.CreateRole:
			role = act.Name
		case actions.SetRolePassword:
			role = act.Name
		default:
			continue
		}
		pw, ok := p.secrets[role]
		if !ok {
			continue
		}
		creds = append(creds, model.Credential{
			Host:     p.Server.Host,
			Port:     p.Server.Port,
			Database: p.credentialDB,
			User:     role,
			Password: pw,
			SSLMode:  p.Server.SSLMode,
		})
	}
	return creds
}
