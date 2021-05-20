package v0

// Schema version 29 adds sector size to miner info

func init() {
	up := batch(`
		ALTER TABLE public.miner_infos ADD COLUMN sector_size bigint NOT NULL;
`)

	down := batch(`
		ALTER TABLE public.miner_infos DROP COLUMN sector_size;
`)

	Patches.MustRegisterTx(up, down)
}
