package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 16 adds power actor claims table

func init() {
	up := batch(`
	CREATE TABLE IF NOT EXISTS "power_actor_claims" (
		"height" bigint not null,
		"miner_id" text not null,
		"state_root" text not null,
		"raw_byte_power" text not null,
		"quality_adj_power" text not null,
		PRIMARY KEY ("height", "miner_id", "state_root")
	);
`)

	down := batch(`
	DROP TABLE IF EXISTS public.power_actor_claims;
`)

	migrations.MustRegisterTx(up, down)
}
