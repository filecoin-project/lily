package migrations

import "github.com/go-pg/migrations/v8"

// Schema version 25 adds Height and ActorName to gas outputs table

func init() {
	up := batch(`

		DO $$
		    BEGIN
				IF (
					SELECT count(*)
					  FROM information_schema.constraint_column_usage
					 WHERE table_schema = 'public'
					   AND table_name = 'derived_gas_outputs'
					    AND constraint_name='derived_gas_outputs_pkey'
					   AND column_name IN ('height','cid','state_root')
				   ) != 3 -- want all three columns in the index
				THEN
					-- Can't change primary key while data exists
					TRUNCATE TABLE public.derived_gas_outputs;

					ALTER TABLE public.derived_gas_outputs ADD COLUMN height bigint NOT NULL;
					ALTER TABLE public.derived_gas_outputs ADD COLUMN actor_name text NOT NULL;
					ALTER TABLE public.derived_gas_outputs DROP CONSTRAINT derived_gas_outputs_pkey;
					ALTER TABLE public.derived_gas_outputs ADD PRIMARY KEY (height,cid,state_root);
				END IF;
			END
		$$;


`)
	down := batch(`
	TRUNCATE TABLE public.derived_gas_outputs;
	ALTER TABLE public.derived_gas_outputs DROP CONSTRAINT derived_gas_outputs_pkey;
	ALTER TABLE public.derived_gas_outputs ADD PRIMARY KEY (cid);
	ALTER TABLE public.derived_gas_outputs DROP COLUMN height;
	ALTER TABLE public.derived_gas_outputs DROP COLUMN actor_name;
`)

	migrations.MustRegisterTx(up, down)
}
