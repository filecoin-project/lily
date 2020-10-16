package migrations

import (
	"github.com/go-pg/migrations/v8"
)

func init() {
	up := batch(`
CREATE TABLE IF NOT EXISTS public.message_parsed (
	"cid" text NOT NULL,
	"from" text NOT NULL,
    "to" text NOT NULL,
	"value" text NOT NULL,
	"method" text NOT NULL,
	"params" jsonb,
	PRIMARY KEY ("cid")
);
`)
	down := batch(`
DROP TABLE IF EXISTS public.message_parsed;
`)

	migrations.MustRegisterTx(up, down)
}
