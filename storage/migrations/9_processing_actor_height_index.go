package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 9 adds a height index to the visor_processing_actors tables

func init() {
	up := batch(`
CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_actors_height" ON "public"."visor_processing_actors" USING BTREE (height DESC);
`)
	down := batch(`
DROP INDEX IF EXISTS "visor_processing_actors_height";
	`)
	migrations.MustRegisterTx(up, down)
}
