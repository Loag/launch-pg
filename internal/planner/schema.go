package planner

import (
	"fmt"

	"launch-pg/internal/model"
	"launch-pg/internal/plan/actions"
)

const publicSchema = "public"

// addSchema prepares the project schema.
//
// "public" differs by version: on PG15+ it is owned by pg_database_owner
// (so the database owner already controls it) and PUBLIC has no CREATE.
// On 13/14 it is owned by the bootstrap superuser and PUBLIC may CREATE;
// only a superuser admin can fix that.
func addSchema(out *Output, db model.DatabaseSpec, owner string, info model.ServerInfo) {
	if db.Schema != publicSchema {
		out.Plan.Add(
			actions.CreateSchema{InDatabase: db.Name, Name: db.Schema, Owner: owner},
			actions.SetSearchPath{Name: db.Name, Schemas: []string{db.Schema, publicSchema}},
		)
		return
	}
	if info.Major() >= 15 {
		return
	}
	if !info.Superuser {
		out.warn(fmt.Sprintf(
			"on Postgres %d schema public is owned by the bootstrap superuser, so any role that can "+
				"connect to %s may create objects in it; use a non-public schema to avoid this",
			info.Major(), db.Name))
		return
	}
	out.Plan.Add(
		actions.SetSchemaOwner{InDatabase: db.Name, Name: publicSchema, Owner: owner},
		actions.GrantSchema{
			InDatabase: db.Name, Schema: publicSchema, Role: actions.Public,
			Privileges: []actions.Privilege{actions.PrivCreate}, Revoke: true,
		},
	)
}
