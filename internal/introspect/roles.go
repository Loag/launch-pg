package introspect

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"launch-pg/internal/model"
	"launch-pg/internal/pg"
)

// Built-in pg_* roles are excluded.
const rolesSQL = `
SELECT r.rolname,
       r.rolcanlogin,
       r.rolsuper,
       r.rolcreatedb,
       r.rolcreaterole,
       r.rolconnlimit,
       COALESCE(array_agg(g.rolname::text ORDER BY g.rolname)
                FILTER (WHERE g.rolname IS NOT NULL), '{}'),
       COALESCE(shobj_description(r.oid, 'pg_authid'), '')
FROM pg_roles r
LEFT JOIN pg_auth_members m ON m.member = r.oid
LEFT JOIN pg_roles g ON g.oid = m.roleid
WHERE r.rolname !~ '^pg_'
GROUP BY r.oid, r.rolname, r.rolcanlogin, r.rolsuper,
         r.rolcreatedb, r.rolcreaterole, r.rolconnlimit
ORDER BY r.rolname`

func (Catalog) Roles(ctx context.Context, db pg.Executor) ([]model.RoleInfo, error) {
	rows, err := db.Query(ctx, rolesSQL)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	roles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.RoleInfo, error) {
		var r model.RoleInfo
		err := row.Scan(&r.Name, &r.CanLogin, &r.Superuser, &r.CreateDB,
			&r.CreateRole, &r.ConnectionLimit, &r.MemberOf, &r.Comment)
		return r, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan roles: %w", err)
	}
	return roles, nil
}
