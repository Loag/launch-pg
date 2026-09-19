package introspect

import (
	"context"
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/pg"
)

const serverInfoSQL = `
SELECT current_setting('server_version_num')::int,
       current_setting('server_version'),
       current_user,
       r.rolsuper,
       r.rolcreatedb,
       r.rolcreaterole
FROM pg_roles r
WHERE r.rolname = current_user`

func (Catalog) ServerInfo(ctx context.Context, db pg.Executor) (model.ServerInfo, error) {
	var info model.ServerInfo
	err := db.QueryRow(ctx, serverInfoSQL).Scan(
		&info.VersionNum,
		&info.Version,
		&info.CurrentUser,
		&info.Superuser,
		&info.CreateDB,
		&info.CreateRole,
	)
	if err != nil {
		return model.ServerInfo{}, fmt.Errorf("read server info: %w", err)
	}
	return info, nil
}
