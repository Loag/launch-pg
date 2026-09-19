// Package pg wraps the Postgres driver behind small interfaces.
package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Executor runs SQL against a single database connection.
type Executor interface {
	Exec(ctx context.Context, sql string, args ...any) error
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Session is an open connection that must be closed.
type Session interface {
	Executor
	Close(ctx context.Context) error
}

// connSession adapts *pgx.Conn to Session.
type connSession struct {
	conn *pgx.Conn
}

func (s *connSession) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := s.conn.Exec(ctx, sql, args...)
	return err
}

func (s *connSession) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return s.conn.Query(ctx, sql, args...)
}

func (s *connSession) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return s.conn.QueryRow(ctx, sql, args...)
}

func (s *connSession) Close(ctx context.Context) error {
	return s.conn.Close(ctx)
}
