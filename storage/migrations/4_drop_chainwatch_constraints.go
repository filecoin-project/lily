package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 4 removes foreign key constraints from the original chainwatch schema which interfere with parallel inserts

func init() {

	up := batch(`
ALTER TABLE public.drand_block_entries DROP CONSTRAINT IF EXISTS block_drand_entries_drand_entries_round_fk;
ALTER TABLE public.drand_block_entries DROP CONSTRAINT IF EXISTS blocks_block_cids_cid_fk;

ALTER TABLE public.blocks_synced DROP CONSTRAINT IF EXISTS blocks_block_cids_cid_fk;

ALTER TABLE public.block_headers DROP CONSTRAINT IF EXISTS blocks_block_cids_cid_fk;

ALTER TABLE public.block_parents DROP CONSTRAINT IF EXISTS blocks_block_cids_cid_fk;

ALTER TABLE public.block_messages DROP CONSTRAINT IF EXISTS blocks_block_cids_cid_fk;

ALTER TABLE public.minerid_dealid_sectorid DROP CONSTRAINT IF EXISTS sectors_sector_ids_id_fk;
ALTER TABLE public.minerid_dealid_sectorid DROP CONSTRAINT IF EXISTS minerid_dealid_sectorid_sector_id_fkey;

ALTER TABLE public.actors DROP CONSTRAINT IF EXISTS id_address_map_actors_id_fk;

ALTER TABLE public.mpool_messages DROP CONSTRAINT IF EXISTS mpool_messages_messages_cid_fk;
`)

	// Note that it is infeasble to add these constraints back after they have been removed since they will almost certainly fail
	// to be met, especially since we do not populate block_cids table any more.
	down := batch("SELECT 1;")

	migrations.MustRegisterTx(up, down)

}
