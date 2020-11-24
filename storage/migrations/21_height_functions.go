package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 21 adds helper functions to convert from unix epoch to fil epoch

func init() {
	up := batch(`
	CREATE OR REPLACE FUNCTION public.unix_to_height(unix_epoch bigint) RETURNS bigint AS $$
		SELECT ((unix_epoch - 1598306400) / 30)::bigint;
	$$ LANGUAGE SQL;

	CREATE OR REPLACE FUNCTION public.height_to_unix(fil_epoch bigint) RETURNS bigint AS $$
		SELECT ((fil_epoch * 30) + 1598306400)::bigint;
	$$ LANGUAGE SQL;
`)

	down := batch(`
	DROP FUNCTION IF EXISTS public.unix_to_height;
	DROP FUNCTION IF EXISTS public.height_to_unix;
`)

	migrations.MustRegisterTx(up, down)
}
