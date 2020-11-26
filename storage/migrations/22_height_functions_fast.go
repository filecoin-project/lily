package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 22 updates adds hints to our helper fns so postgres can optimise them.

// IMMUTABLE - the function cannot modify the database and always returns the same result when given the same argument value
// RETURNS NULL ON NULL INPUT - the function is not executed when there are null arguments; instead a null result is assumed automatically.
// PARALLEL SAFE - safe to run in parallel mode without restriction.

func init() {
	up := batch(`
	CREATE OR REPLACE FUNCTION public.unix_to_height(unix_epoch bigint) RETURNS bigint AS $$
		SELECT ((unix_epoch - 1598306400) / 30)::bigint;
	$$ LANGUAGE SQL IMMUTABLE RETURNS NULL ON NULL INPUT PARALLEL SAFE;

	CREATE OR REPLACE FUNCTION public.height_to_unix(fil_epoch bigint) RETURNS bigint AS $$
		SELECT ((fil_epoch * 30) + 1598306400)::bigint;
	$$ LANGUAGE SQL IMMUTABLE RETURNS NULL ON NULL INPUT PARALLEL SAFE;
`)

	down := batch(`
	DROP FUNCTION IF EXISTS public.unix_to_height;
	DROP FUNCTION IF EXISTS public.height_to_unix;
`)

	migrations.MustRegisterTx(up, down)
}
