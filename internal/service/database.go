package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"launch-pg/internal/backup"
	"launch-pg/internal/model"
	"launch-pg/internal/planner"
)

// DatabaseService lists, clones and drops databases.
type DatabaseService struct {
	state      *StateLoader
	planner    *planner.Planner
	dumper     backup.Dumper
	backupsDir string
	now        func() time.Time
}

func NewDatabaseService(
	state *StateLoader,
	planner *planner.Planner,
	dumper backup.Dumper,
	backupsDir string,
	now func() time.Time,
) *DatabaseService {
	return &DatabaseService{state: state, planner: planner, dumper: dumper, backupsDir: backupsDir, now: now}
}

// List returns non-template databases.
func (s *DatabaseService) List(ctx context.Context, server string) ([]model.DatabaseInfo, error) {
	_, state, err := s.state.Load(ctx, server)
	if err != nil {
		return nil, err
	}
	var out []model.DatabaseInfo
	for _, d := range state.Databases {
		if !d.IsTemplate {
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *DatabaseService) PrepareClone(ctx context.Context, server string, c planner.Clone) (Prepared, error) {
	srv, state, err := s.state.Load(ctx, server)
	if err != nil {
		return Prepared{}, err
	}
	out, err := s.planner.CloneDatabase(c, state)
	if err != nil {
		return Prepared{}, err
	}
	return newPrepared(srv, out, c.Target), nil
}

type DropRequest struct {
	Server   string
	Teardown planner.Teardown
	// Backup requests a pg_dump first; BackupPath "" means the default
	// location under the state directory.
	Backup     bool
	BackupPath string
}

func (s *DatabaseService) PrepareDrop(ctx context.Context, req DropRequest) (Prepared, error) {
	srv, state, err := s.state.Load(ctx, req.Server)
	if err != nil {
		return Prepared{}, err
	}
	if req.Teardown.Database == srv.MaintenanceDB {
		return Prepared{}, fmt.Errorf("refusing to drop %s: it is launch-pg's maintenance database for this server",
			srv.MaintenanceDB)
	}
	out, err := s.planner.DropDatabase(req.Teardown, state)
	if err != nil {
		return Prepared{}, err
	}
	prepared := newPrepared(srv, out, "")

	if req.Backup {
		if err := s.dumper.Check(ctx, state.Info.Major()); err != nil {
			return Prepared{}, err
		}
		path := req.BackupPath
		if path == "" {
			path = filepath.Join(s.backupsDir, fmt.Sprintf("%s-%s-%s.dump",
				srv.Name, req.Teardown.Database, s.now().UTC().Format("20060102T150405Z")))
		}
		prepared.Backup = &BackupStep{Database: req.Teardown.Database, Path: path}
	}
	return prepared, nil
}
