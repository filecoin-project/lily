package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 29 fixes https://github.com/filecoin-project/sentinel-visor/issues/459

func init() {
	up := batch(`
	ALTER TABLE public.multisig_transactions ALTER COLUMN params DROP NOT NULL;
`)

	down := batch(`
	ALTER TABLE public.multisig_transactions ALTER COLUMN params SET NOT NULL;
`)

	migrations.MustRegisterTx(up, down)
}
