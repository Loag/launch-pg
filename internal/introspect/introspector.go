// Package introspect reads the current state of a Postgres server from its catalogs.
package introspect

import (
	"context"

	"launch-pg/internal/model"
	"launch-pg/internal/pg"
)

// Introspector reads server state through an open session.
type Introspector interface {
	ServerInfo(ctx context.Context, db pg.Executor) (model.ServerInfo, error)
	Databases(ctx context.Context, db pg.Executor) ([]model.DatabaseInfo, error)
	Roles(ctx context.Context, db pg.Executor) ([]model.RoleInfo, error)
}

// Catalog implements Introspector with queries against pg_catalog.
type Catalog struct{}

func NewCatalog() *Catalog { return &Catalog{} }
