package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 23 fixes the miner sector deals primary key

func init() {
	up := batch(`
		-- Only run this destructive migration if the constraint doesn't exist
		DO $$
		    BEGIN
				IF (
					SELECT count(*)
					  FROM information_schema.constraint_column_usage
					 WHERE table_schema = 'public'
					   AND table_name = 'miner_sector_deals'
					    AND constraint_name='miner_sector_deals_pkey'
					   AND column_name IN ('height','miner_id','sector_id','deal_id')
				   ) != 4

				THEN

					-- Can't change primary key while data exists
					TRUNCATE TABLE public.miner_sector_deals;

					-- old index from when table was named miner_deal_sectors
					ALTER TABLE public.miner_sector_deals DROP CONSTRAINT IF EXISTS miner_deal_sectors_pkey;

					ALTER TABLE public.miner_sector_deals DROP CONSTRAINT IF EXISTS miner_sector_deals_pkey;
				   	ALTER TABLE public.miner_sector_deals ADD PRIMARY KEY (height, miner_id, sector_id, deal_id);

				END IF;
			END
		$$;
`)

	down := batch(`
		-- don't recreate the buggy constraint
		SELECT 1;
`)

	migrations.MustRegisterTx(up, down)
}
