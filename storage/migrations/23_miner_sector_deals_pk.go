package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 23 fixes the miner sector deals primary key

func init() {
	up := batch(`
		-- old index from when table was named miner_deal_sectors
		ALTER TABLE public.miner_sector_deals DROP CONSTRAINT IF EXISTS miner_deal_sectors_pkey;

		ALTER TABLE public.miner_sector_deals DROP CONSTRAINT IF EXISTS miner_sector_deals_pkey;
	   	ALTER TABLE public.miner_sector_deals ADD PRIMARY KEY (height, miner_id, sector_id, deal_id);
`)

	down := batch(`
		ALTER TABLE public.miner_sector_deals DROP CONSTRAINT IF EXISTS miner_sector_deals_pkey;
	   	ALTER TABLE public.miner_sector_deals ADD PRIMARY KEY (height, miner_id, sector_id);
`)

	migrations.MustRegisterTx(up, down)
}
