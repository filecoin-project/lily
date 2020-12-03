package migrations

import "github.com/go-pg/migrations/v8"

// Schema version 26 converts string types containing numbers to numeric types.

func init() {
	up := batch(`
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN value TYPE numeric USING (value::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN gas_fee_cap TYPE numeric USING (gas_fee_cap::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN gas_premium TYPE numeric USING (gas_premium::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN parent_base_fee TYPE numeric USING (parent_base_fee::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN base_fee_burn TYPE numeric USING (base_fee_burn::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN over_estimation_burn TYPE numeric USING (over_estimation_burn::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN miner_penalty TYPE numeric USING (miner_penalty::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN miner_tip TYPE numeric USING (miner_tip::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN refund TYPE numeric USING (refund::numeric);

    ALTER TABLE public.messages ALTER COLUMN value TYPE numeric USING (value::numeric);
    ALTER TABLE public.messages ALTER COLUMN gas_fee_cap TYPE numeric USING (gas_fee_cap::numeric);
    ALTER TABLE public.messages ALTER COLUMN gas_premium TYPE numeric USING (gas_premium::numeric);

    ALTER TABLE public.miner_locked_funds ALTER COLUMN locked_funds TYPE numeric USING (locked_funds::numeric);
    ALTER TABLE public.miner_locked_funds ALTER COLUMN initial_pledge TYPE numeric USING (initial_pledge::numeric);
    ALTER TABLE public.miner_locked_funds ALTER COLUMN pre_commit_deposits TYPE numeric USING (pre_commit_deposits::numeric);

    ALTER TABLE public.miner_fee_debts ALTER COLUMN fee_debt TYPE numeric USING (fee_debt::numeric);

    ALTER TABLE public.miner_pre_commit_infos ALTER COLUMN pre_commit_deposit TYPE numeric USING (pre_commit_deposit::numeric);
    ALTER TABLE public.miner_pre_commit_infos ALTER COLUMN deal_weight TYPE numeric USING (deal_weight::numeric);
    ALTER TABLE public.miner_pre_commit_infos ALTER COLUMN verified_deal_weight TYPE numeric USING (verified_deal_weight::numeric);

    ALTER TABLE public.miner_sector_infos ALTER COLUMN deal_weight TYPE numeric USING (deal_weight::numeric);
    ALTER TABLE public.miner_sector_infos ALTER COLUMN verified_deal_weight TYPE numeric USING (verified_deal_weight::numeric);
    ALTER TABLE public.miner_sector_infos ALTER COLUMN initial_pledge TYPE numeric USING (initial_pledge::numeric);
    ALTER TABLE public.miner_sector_infos ALTER COLUMN expected_day_reward TYPE numeric USING (expected_day_reward::numeric);
    ALTER TABLE public.miner_sector_infos ALTER COLUMN expected_storage_pledge TYPE numeric USING (expected_storage_pledge::numeric);

    ALTER TABLE public.chain_powers ALTER COLUMN total_pledge_collateral TYPE numeric USING (total_pledge_collateral::numeric);
    ALTER TABLE public.chain_powers ALTER COLUMN total_raw_bytes_power TYPE numeric USING (total_raw_bytes_power::numeric);
    ALTER TABLE public.chain_powers ALTER COLUMN total_raw_bytes_committed TYPE numeric USING (total_raw_bytes_committed::numeric);
    ALTER TABLE public.chain_powers ALTER COLUMN total_qa_bytes_power TYPE numeric USING (total_qa_bytes_power::numeric);
    ALTER TABLE public.chain_powers ALTER COLUMN total_qa_bytes_committed TYPE numeric USING (total_qa_bytes_committed::numeric);
    ALTER TABLE public.chain_powers ALTER COLUMN qa_smoothed_position_estimate TYPE numeric USING (qa_smoothed_position_estimate::numeric);
    ALTER TABLE public.chain_powers ALTER COLUMN qa_smoothed_velocity_estimate TYPE numeric USING (qa_smoothed_velocity_estimate::numeric);

    ALTER TABLE public.chain_rewards ALTER COLUMN effective_baseline_power TYPE numeric USING (effective_baseline_power::numeric);
    ALTER TABLE public.chain_rewards ALTER COLUMN cum_sum_baseline TYPE numeric USING (cum_sum_baseline::numeric);
    ALTER TABLE public.chain_rewards ALTER COLUMN cum_sum_realized TYPE numeric USING (cum_sum_realized::numeric);
    ALTER TABLE public.chain_rewards ALTER COLUMN new_baseline_power TYPE numeric USING (new_baseline_power::numeric);
    ALTER TABLE public.chain_rewards ALTER COLUMN new_reward_smoothed_position_estimate TYPE numeric USING (new_reward_smoothed_position_estimate::numeric);
    ALTER TABLE public.chain_rewards ALTER COLUMN new_reward_smoothed_velocity_estimate TYPE numeric USING (new_reward_smoothed_velocity_estimate::numeric);
    ALTER TABLE public.chain_rewards ALTER COLUMN total_mined_reward TYPE numeric USING (total_mined_reward::numeric);
    ALTER TABLE public.chain_rewards ALTER COLUMN new_reward TYPE numeric USING (new_reward::numeric);

    ALTER TABLE public.message_gas_economy ALTER COLUMN base_fee TYPE numeric USING (base_fee::numeric);
    ALTER TABLE public.message_gas_economy ALTER COLUMN gas_limit_total TYPE numeric USING (base_fee::numeric);
    ALTER TABLE public.message_gas_economy ALTER COLUMN gas_limit_unique_total TYPE numeric USING (base_fee::numeric);

	-- cannot modify column type if a view depends on it.
	DROP VIEW IF EXISTS chain_visualizer_chain_data_view;

	ALTER TABLE public.power_actor_claims ALTER COLUMN raw_byte_power TYPE numeric USING (raw_byte_power::numeric);
    ALTER TABLE public.power_actor_claims ALTER COLUMN quality_adj_power TYPE numeric USING (quality_adj_power::numeric);

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
`)
	down := batch(`
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN value TYPE text USING (value::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN gas_fee_cap TYPE text USING (gas_fee_cap::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN gas_premium TYPE text USING (gas_premium::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN parent_base_fee TYPE text USING (parent_base_fee::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN base_fee_burn TYPE text USING (base_fee_burn::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN over_estimation_burn TYPE text USING (over_estimation_burn::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN miner_penalty TYPE text USING (miner_penalty::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN miner_tip TYPE text USING (miner_tip::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN refund TYPE text USING (refund::text);

    ALTER TABLE public.messages ALTER COLUMN value TYPE text USING (value::text);
    ALTER TABLE public.messages ALTER COLUMN gas_fee_cap TYPE text USING (gas_fee_cap::text);
    ALTER TABLE public.messages ALTER COLUMN gas_premium TYPE text USING (gas_premium::text);

    ALTER TABLE public.miner_locked_funds ALTER COLUMN locked_funds TYPE text USING (locked_funds::text);
    ALTER TABLE public.miner_locked_funds ALTER COLUMN initial_pledge TYPE text USING (initial_pledge::text);
    ALTER TABLE public.miner_locked_funds ALTER COLUMN pre_commit_deposits TYPE text USING (pre_commit_deposits::text);

    ALTER TABLE public.miner_fee_debts ALTER COLUMN fee_debt TYPE text USING (fee_debt::text);

    ALTER TABLE public.miner_pre_commit_infos ALTER COLUMN pre_commit_deposit TYPE text USING (pre_commit_deposit::text);
    ALTER TABLE public.miner_pre_commit_infos ALTER COLUMN deal_weight TYPE text USING (deal_weight::text);
    ALTER TABLE public.miner_pre_commit_infos ALTER COLUMN verified_deal_weight TYPE text USING (verified_deal_weight::text);

    ALTER TABLE public.miner_sector_infos ALTER COLUMN deal_weight TYPE text USING (deal_weight::text);
    ALTER TABLE public.miner_sector_infos ALTER COLUMN verified_deal_weight TYPE text USING (verified_deal_weight::text);
    ALTER TABLE public.miner_sector_infos ALTER COLUMN initial_pledge TYPE text USING (initial_pledge::text);
    ALTER TABLE public.miner_sector_infos ALTER COLUMN expected_day_reward TYPE text USING (expected_day_reward::text);
    ALTER TABLE public.miner_sector_infos ALTER COLUMN expected_storage_pledge TYPE text USING (expected_storage_pledge::text);

    ALTER TABLE public.chain_powers ALTER COLUMN total_pledge_collateral TYPE text USING (total_pledge_collateral::text);
    ALTER TABLE public.chain_powers ALTER COLUMN total_raw_bytes_power TYPE text USING (total_raw_bytes_power::text);
    ALTER TABLE public.chain_powers ALTER COLUMN total_raw_bytes_committed TYPE text USING (total_raw_bytes_committed::text);
    ALTER TABLE public.chain_powers ALTER COLUMN total_qa_bytes_power TYPE text USING (total_qa_bytes_power::text);
    ALTER TABLE public.chain_powers ALTER COLUMN total_qa_bytes_committed TYPE text USING (total_qa_bytes_committed::text);

    ALTER TABLE public.chain_rewards ALTER COLUMN effective_baseline_power TYPE text USING (effective_baseline_power::text);
    ALTER TABLE public.chain_rewards ALTER COLUMN cum_sum_baseline TYPE text USING (cum_sum_baseline::text);
    ALTER TABLE public.chain_rewards ALTER COLUMN cum_sum_realized TYPE text USING (cum_sum_realized::text);
    ALTER TABLE public.chain_rewards ALTER COLUMN new_baseline_power TYPE text USING (new_baseline_power::text);
    ALTER TABLE public.chain_rewards ALTER COLUMN new_reward_smoothed_position_estimate TYPE text USING (new_reward_smoothed_position_estimate::text);
    ALTER TABLE public.chain_rewards ALTER COLUMN new_reward_smoothed_velocity_estimate TYPE text USING (new_reward_smoothed_velocity_estimate::text);
    ALTER TABLE public.chain_rewards ALTER COLUMN total_mined_reward TYPE text USING (total_mined_reward::text);
    ALTER TABLE public.chain_rewards ALTER COLUMN new_reward TYPE text USING (new_reward::text);

    ALTER TABLE public.message_gas_economy ALTER COLUMN base_fee TYPE float USING (base_fee::float);
    ALTER TABLE public.message_gas_economy ALTER COLUMN gas_limit_total TYPE text USING (base_fee::text);
    ALTER TABLE public.message_gas_economy ALTER COLUMN gas_limit_unique_total TYPE text USING (base_fee::text);

	-- cannot modify column type if a view depends on it.
	DROP VIEW IF EXISTS chain_visualizer_chain_data_view;

    ALTER TABLE public.power_actor_claims ALTER COLUMN raw_byte_power TYPE text USING (raw_byte_power::text);
    ALTER TABLE public.power_actor_claims ALTER COLUMN quality_adj_power TYPE text USING (quality_adj_power::text);

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
`)

	migrations.MustRegisterTx(up, down)
}
