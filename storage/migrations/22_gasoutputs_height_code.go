package migrations

import "github.com/go-pg/migrations/v8"

// Schema version 22 adds Height and ActorName to gas outputs table

func init() {
	up := batch(`
	ALTER TABLE public.derived_gas_outputs ADD COLUMN height int8;
	ALTER TABLE public.derived_gas_outputs ADD COLUMN actor_name text;
`)
	down := batch(`
	ALTER TABLE public.derived_gas_outputs DROP COLUMN height;
	ALTER TABLE public.derived_gas_outputs DROP COLUMN actor_name;
`)

	migrations.MustRegisterTx(up, down)
}
