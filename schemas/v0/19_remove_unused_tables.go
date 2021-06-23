package v0

// Schema version 19 drops unused tables.

func init() {
	up := batch(`
DROP VIEW IF EXISTS chain_visualizer_chain_data_view;
CREATE VIEW chain_visualizer_chain_data_view AS
	SELECT
		main_block.cid AS block,
		bp.parent AS parent,
		main_block.miner,
		main_block.height,
		main_block.parent_weight AS parentweight,
		main_block.timestamp,
		main_block.parent_state_root AS parentstateroot,
		parent_block.timestamp AS parenttimestamp,
		parent_block.height AS parentheight,
		-- was miner_power.raw_bytes_power (plural bytes)
		pac.raw_byte_power AS parentpower,
		-- was blocks_synced.synced_at
		main_block.timestamp AS syncedtimestamp,
		(SELECT COUNT(*) FROM block_messages WHERE block_messages.block = main_block.cid) AS messages
	FROM
		block_headers main_block
	LEFT JOIN
		block_parents bp ON bp.block = main_block.cid
	LEFT JOIN
		block_headers parent_block ON parent_block.cid = bp.parent
	LEFT JOIN
		-- was miner_power (singular)
		power_actor_claims pac ON main_block.parent_state_root = pac.state_root
;

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

	patches.MustRegisterTx(up, down)
}
