package plan

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"launch-pg/internal/audit"
	"launch-pg/internal/config"
	"launch-pg/internal/pg"
)

// fakeAction is a minimal Action for engine tests.
type fakeAction struct {
	name  string
	db    string
	nonTx bool
	sql   []Statement
	drops string
}

func (f fakeAction) Kind() string        { return "fake" }
func (f fakeAction) Describe() string    { return f.name }
func (f fakeAction) Database() string    { return f.db }
func (f fakeAction) Transactional() bool { return !f.nonTx }
func (f fakeAction) Validate() error     { return nil }
func (f fakeAction) Statements() []Statement {
	if f.sql != nil {
		return f.sql
	}
	return []Statement{Stmt("-- " + f.name)}
}

type droppingAction struct{ fakeAction }

func (d droppingAction) DroppedDatabase() string { return d.drops }

// fakeConnector records every statement as "db: sql" and fails any
// statement containing failOn.
type fakeConnector struct {
	log    []string
	failOn string
	closed []string
}

func (c *fakeConnector) Connect(_ context.Context, _ config.Server, db string) (pg.Session, error) {
	return &fakeSession{c: c, db: db}, nil
}

type fakeSession struct {
	c  *fakeConnector
	db string
}

func (s *fakeSession) Exec(_ context.Context, sql string, _ ...any) error {
	s.c.log = append(s.c.log, s.db+": "+sql)
	if s.c.failOn != "" && strings.Contains(sql, s.c.failOn) {
		return errors.New("boom")
	}
	return nil
}
func (s *fakeSession) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (s *fakeSession) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (s *fakeSession) Close(context.Context) error {
	s.c.closed = append(s.c.closed, s.db)
	return nil
}

type memAudit struct{ events []audit.Event }

func (m *memAudit) Record(e audit.Event) error {
	m.events = append(m.events, e)
	return nil
}
