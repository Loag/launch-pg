package plan

import (
	"context"
	"errors"

	"launch-pg/internal/config"
	"launch-pg/internal/pg"
)

// sessionPool lazily opens one session per database for a single Apply.
type sessionPool struct {
	connector pg.Connector
	server    config.Server
	open      map[string]pg.Session
}

func newSessionPool(connector pg.Connector, server config.Server) *sessionPool {
	return &sessionPool{connector: connector, server: server, open: map[string]pg.Session{}}
}

func (p *sessionPool) get(ctx context.Context, database string) (pg.Session, error) {
	if database == "" {
		database = p.server.MaintenanceDB
	}
	if s, ok := p.open[database]; ok {
		return s, nil
	}
	s, err := p.connector.Connect(ctx, p.server, database)
	if err != nil {
		return nil, err
	}
	p.open[database] = s
	return s, nil
}

// close drops the session for a database, e.g. before dropping it.
func (p *sessionPool) close(ctx context.Context, database string) error {
	s, ok := p.open[database]
	if !ok {
		return nil
	}
	delete(p.open, database)
	return s.Close(ctx)
}

func (p *sessionPool) closeAll(ctx context.Context) error {
	var errs []error
	for db, s := range p.open {
		errs = append(errs, s.Close(ctx))
		delete(p.open, db)
	}
	return errors.Join(errs...)
}
