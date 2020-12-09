package migrations

import "github.com/go-pg/migrations/v8"

// Schema version 27 defines a visor schema.

func init() {
	up := batch(`
	-- create the new schema
	CREATE SCHEMA visor;

	-- flip, I mean move tables
	ALTER TABLE public.actor_states SET SCHEMA visor;
	ALTER TABLE public.actors SET SCHEMA visor;
	ALTER TABLE public.block_headers SET SCHEMA visor;
	ALTER TABLE public.block_messages SET SCHEMA visor;
	ALTER TABLE public.block_parents SET SCHEMA visor;
	ALTER TABLE public.chain_economics SET SCHEMA visor;
	ALTER TABLE public.chain_powers SET SCHEMA visor;
	ALTER TABLE public.chain_rewards SET SCHEMA visor;
	ALTER TABLE public.derived_gas_outputs SET SCHEMA visor;
	ALTER TABLE public.drand_block_entries SET SCHEMA visor;
	ALTER TABLE public.id_address_map SET SCHEMA visor;
	ALTER TABLE public.id_addresses SET SCHEMA visor;
	ALTER TABLE public.market_deal_proposals SET SCHEMA visor;
	ALTER TABLE public.market_deal_states SET SCHEMA visor;
	ALTER TABLE public.message_gas_economy SET SCHEMA visor;
	ALTER TABLE public.messages SET SCHEMA visor;
	ALTER TABLE public.miner_current_deadline_infos SET SCHEMA visor;
	ALTER TABLE public.miner_fee_debts SET SCHEMA visor;
	ALTER TABLE public.miner_infos SET SCHEMA visor;
	ALTER TABLE public.miner_locked_funds SET SCHEMA visor;
	ALTER TABLE public.miner_pre_commit_infos SET SCHEMA visor;
	ALTER TABLE public.miner_sector_deals SET SCHEMA visor;
	ALTER TABLE public.miner_sector_events SET SCHEMA visor;
	ALTER TABLE public.miner_sector_infos SET SCHEMA visor;
	ALTER TABLE public.miner_sector_posts SET SCHEMA visor;
	ALTER TABLE public.multisig_transactions SET SCHEMA visor;
	ALTER TABLE public.parsed_messages SET SCHEMA visor;
	ALTER TABLE public.power_actor_claims SET SCHEMA visor;
	ALTER TABLE public.receipts SET SCHEMA visor;
	ALTER TABLE public.sector_precommit_info SET SCHEMA visor;
	ALTER TABLE public.visor_processing_actors SET SCHEMA visor;
	ALTER TABLE public.visor_processing_messages SET SCHEMA visor;
	ALTER TABLE public.visor_processing_reports SET SCHEMA visor;
	ALTER TABLE public.visor_processing_stats SET SCHEMA visor;
	ALTER TABLE public.visor_processing_tipsets SET SCHEMA visor;

	-- move types to visor
	ALTER TYPE public.miner_sector_event_type SET SCHEMA visor;

	-- move views to visor
	ALTER VIEW public.chain_visualizer_blocks_view SET SCHEMA visor;
	ALTER VIEW public.chain_visualizer_blocks_with_parents_view SET SCHEMA visor;
	ALTER VIEW public.chain_visualizer_chain_data_view SET SCHEMA visor;
	ALTER VIEW public.chain_visualizer_orphans_view SET SCHEMA visor;

	-- move materialized views to visor
	ALTER MATERIALIZED VIEW public.derived_consensus_chain_view SET SCHEMA visor;
	ALTER MATERIALIZED VIEW public.state_heights SET SCHEMA visor;

	-- move routines to visor
	ALTER ROUTINE public.actor_tips(epoch bigint) SET SCHEMA visor;
	ALTER ROUTINE public.unix_to_height(unix_epoch bigint) SET SCHEMA visor;
	ALTER ROUTINE public.height_to_unix(fil_epoch bigint) SET SCHEMA visor;
`)
	down := batch(`
	-- move tables back to public
	ALTER TABLE visor.actor_states SET SCHEMA public;
	ALTER TABLE visor.actors SET SCHEMA public;
	ALTER TABLE visor.block_headers SET SCHEMA public;
	ALTER TABLE visor.block_messages SET SCHEMA public;
	ALTER TABLE visor.block_parents SET SCHEMA public;
	ALTER TABLE visor.chain_economics SET SCHEMA public;
	ALTER TABLE visor.chain_powers SET SCHEMA public;
	ALTER TABLE visor.chain_rewards SET SCHEMA public;
	ALTER TABLE visor.derived_gas_outputs SET SCHEMA public;
	ALTER TABLE visor.drand_block_entries SET SCHEMA public;
	ALTER TABLE visor.id_address_map SET SCHEMA public;
	ALTER TABLE visor.id_addresses SET SCHEMA public;
	ALTER TABLE visor.market_deal_proposals SET SCHEMA public;
	ALTER TABLE visor.market_deal_states SET SCHEMA public;
	ALTER TABLE visor.message_gas_economy SET SCHEMA public;
	ALTER TABLE visor.messages SET SCHEMA public;
	ALTER TABLE visor.miner_current_deadline_infos SET SCHEMA public;
	ALTER TABLE visor.miner_fee_debts SET SCHEMA public;
	ALTER TABLE visor.miner_infos SET SCHEMA public;
	ALTER TABLE visor.miner_locked_funds SET SCHEMA public;
	ALTER TABLE visor.miner_pre_commit_infos SET SCHEMA public;
	ALTER TABLE visor.miner_sector_deals SET SCHEMA public;
	ALTER TABLE visor.miner_sector_events SET SCHEMA public;
	ALTER TABLE visor.miner_sector_infos SET SCHEMA public;
	ALTER TABLE visor.miner_sector_posts SET SCHEMA public;
	ALTER TABLE visor.multisig_transactions SET SCHEMA public;
	ALTER TABLE visor.parsed_messages SET SCHEMA public;
	ALTER TABLE visor.power_actor_claims SET SCHEMA public;
	ALTER TABLE visor.receipts SET SCHEMA public;
	ALTER TABLE visor.sector_precommit_info SET SCHEMA public;
	ALTER TABLE visor.visor_processing_actors SET SCHEMA public;
	ALTER TABLE visor.visor_processing_messages SET SCHEMA public;
	ALTER TABLE visor.visor_processing_reports SET SCHEMA public;
	ALTER TABLE visor.visor_processing_stats SET SCHEMA public;
	ALTER TABLE visor.visor_processing_tipsets SET SCHEMA public;

	-- move types back to public
	ALTER TYPE visor.miner_sector_event_type SET SCHEMA public;

	-- move views back to public
	ALTER VIEW visor.chain_visualizer_blocks_view SET SCHEMA public;
	ALTER VIEW visor.chain_visualizer_blocks_with_parents_view SET SCHEMA public;
	ALTER VIEW visor.chain_visualizer_chain_data_view SET SCHEMA public;
	ALTER VIEW visor.chain_visualizer_orphans_view SET SCHEMA public;

	-- move materialized views back to public
	ALTER MATERIALIZED VIEW visor.derived_consensus_chain_view SET SCHEMA public;
	ALTER MATERIALIZED VIEW visor.state_heights SET SCHEMA public;

	-- move routines back to public
	ALTER ROUTINE visor.actor_tips(epoch bigint) SET SCHEMA public;
	ALTER ROUTINE visor.unix_to_height(unix_epoch bigint) SET SCHEMA public;
	ALTER ROUTINE visor.height_to_unix(fil_epoch bigint) SET SCHEMA public;

	-- remove the visor schema
	DROP SCHEMA visor;
`)

	migrations.MustRegisterTx(up, down)
}
