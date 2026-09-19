package introspect

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"launch-pg/internal/model"
	"launch-pg/internal/pg"
)

// Size is only computed where the admin can CONNECT; otherwise
// pg_database_size raises a permission error. ACL entries are encoded as
// "grantee=PRIVILEGE" (grantee 0 is PUBLIC) and split on the last '='.
const databasesSQL = `
SELECT d.datname,
       pg_get_userbyid(d.datdba),
       pg_encoding_to_char(d.encoding),
       CASE WHEN has_database_privilege(d.oid, 'CONNECT')
            THEN pg_database_size(d.oid) END,
       d.datistemplate,
       d.datallowconn,
       has_database_privilege(d.oid, 'CONNECT'),
       COALESCE(shobj_description(d.oid, 'pg_database'), ''),
       d.datacl IS NULL,
       COALESCE((SELECT array_agg(
                    CASE WHEN a.grantee = 0 THEN 'PUBLIC'
                         ELSE pg_get_userbyid(a.grantee)::text END
                    || '=' || a.privilege_type)
                 FROM aclexplode(d.datacl) a), '{}'),
       (SELECT count(*) FROM pg_stat_activity s WHERE s.datid = d.oid)::int
FROM pg_database d
ORDER BY d.datname`

func (Catalog) Databases(ctx context.Context, db pg.Executor) ([]model.DatabaseInfo, error) {
	rows, err := db.Query(ctx, databasesSQL)
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}
	dbs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.DatabaseInfo, error) {
		var (
			d   model.DatabaseInfo
			acl []string
		)
		err := row.Scan(&d.Name, &d.Owner, &d.Encoding, &d.SizeBytes, &d.IsTemplate,
			&d.AllowConn, &d.CanConnect, &d.Comment, &d.DefaultACL, &acl, &d.Connections)
		d.ACL = parseACL(acl)
		return d, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan databases: %w", err)
	}
	return dbs, nil
}

func parseACL(encoded []string) []model.ACLEntry {
	out := make([]model.ACLEntry, 0, len(encoded))
	for _, e := range encoded {
		i := strings.LastIndex(e, "=")
		if i < 0 {
			continue
		}
		out = append(out, model.ACLEntry{Grantee: e[:i], Privilege: e[i+1:]})
	}
	return out
}
