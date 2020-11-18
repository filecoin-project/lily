package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 19 drops unused tables.

func init() {
	up := batch(`
	DROP MATERIALIZED VIEW IF EXISTS public.chain_visualizer_chain_data_view; 		-- artifact from chainwatch
	DROP MATERIALIZED VIEW IF EXISTS public.top_miners_by_base_reward; 				-- artifact from chainwatch
	DROP MATERIALIZED VIEW IF EXISTS public.top_miners_by_base_reward_max_height;	-- artifact from chainwatch


	DROP TABLE IF EXISTS public.block_cids; 				-- artifact from chainwatch
	DROP TABLE IF EXISTS public.block_synced; 				-- artifact from chainwatch
	DROP TABLE IF EXISTS public.mpool_messages; 			-- artifact from chainwatch
	DROP TABLE IF EXISTS public.chain_power; 				-- replaced by chain_powers
	DROP TABLE IF EXISTS public.chain_reward; 				-- replaces by chain_rewards
	DROP TABLE IF EXISTS public.ip_address_map; 			-- replaced by ip_addresses
	DROP TABLE IF EXISTS public.miner_info; 				-- replaced by miner_infos
	DROP TABLE IF EXISTS public.miner_states; 				-- replaced by miner_infos
	DROP TABLE IF EXISTS public.miner_power; 				-- replaced by power_actor_claims
	DROP TABLE IF EXISTS public.miner_powers; 				-- replaced by power_actor_claims
	DROP TABLE IF EXISTS public.minerid_dealid_sectorid; 	-- replaced by miner_sector_deals
	DROP TABLE IF EXISTS public.sector_info; 				-- replaced by miner_sector_infos
	DROP TABLE IF EXISTS public.sector_precommit_infos; 	-- replaced by miner_pre_commit_infos

`)

	// no take backsies.
	down := batch(`SELECT 1;`)

	migrations.MustRegisterTx(up, down)
}
