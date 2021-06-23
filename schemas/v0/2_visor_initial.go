package v0

// Schema version 2 is the initial schema used by visor

func init() {
	up := batch(`
ALTER TABLE public.actors RENAME COLUMN stateroot TO state_root;
ALTER TABLE public.actors ALTER nonce type bigint;

ALTER TABLE public.actor_states ALTER COLUMN state SET DATA TYPE jsonb using state::JSONB;

ALTER TABLE public.blocks RENAME TO block_headers;
ALTER TABLE public.block_headers RENAME COLUMN forksig TO fork_signaling;
ALTER TABLE public.block_headers RENAME COLUMN parentstateroot TO parent_state_root;
ALTER TABLE public.block_headers RENAME COLUMN parentweight TO parent_weight;
ALTER TABLE public.block_headers ALTER COLUMN parent_weight TYPE text USING (parent_weight::text);

ALTER TABLE public.blocks_synced ADD COLUMN height bigint;
ALTER TABLE public.blocks_synced ADD COLUMN completed_at timestamptz;

ALTER TABLE public.blocks_synced
ALTER synced_at TYPE TIMESTAMP WITH TIME ZONE
USING to_timestamp(synced_at) AT TIME ZONE 'UTC';

ALTER TABLE public.blocks_synced
ALTER processed_at TYPE TIMESTAMP WITH TIME ZONE
USING to_timestamp(processed_at) AT TIME ZONE 'UTC';


ALTER TABLE public.block_drand_entries RENAME TO drand_block_entries;

CREATE TABLE IF NOT EXISTS "chain_powers" (
	"state_root" text,
	"new_raw_bytes_power" text NOT NULL,
	"new_qa_bytes_power" text NOT NULL,
	"new_pledge_collateral" text NOT NULL,
	"total_raw_bytes_power" text NOT NULL,
	"total_raw_bytes_committed" text NOT NULL,
	"total_qa_bytes_power" text NOT NULL,
	"total_qa_bytes_committed" text NOT NULL,
	"total_pledge_collateral" text NOT NULL,
	"qa_smoothed_position_estimate" text NOT NULL,
	"qa_smoothed_velocity_estimate" text NOT NULL,
	"miner_count" bigint,
	"minimum_consensus_miner_count" bigint,
	PRIMARY KEY ("state_root")
);

CREATE TABLE IF NOT EXISTS "chain_rewards" (
	"state_root" text NOT NULL,
	"cum_sum_baseline" text NOT NULL,
	"cum_sum_realized" text NOT NULL,
	"effective_baseline_power" text NOT NULL,
	"new_baseline_power" text NOT NULL,
	"new_reward_smoothed_position_estimate" text NOT NULL,
	"new_reward_smoothed_velocity_estimate" text NOT NULL,
	"total_mined_reward" text NOT NULL,
	"new_reward" text,
	"effective_network_time" bigint,
	PRIMARY KEY ("state_root")
);


CREATE TABLE IF NOT EXISTS "id_addresses" (
	"id" text NOT NULL,
	"address" text NOT NULL,
	"state_root" text NOT NULL,
	PRIMARY KEY ("id", "address", "state_root")
);

ALTER TABLE public.market_deal_proposals ADD COLUMN label text;

-- TODO: check if slashed_epoch should be removed
-- ALTER TABLE public.market_deal_proposals DROP COLUMN slashed_epoch;

CREATE TABLE IF NOT EXISTS "miner_deal_sectors" (
	"miner_id" text NOT NULL,
	"sector_id" bigserial,
	"deal_id" bigint,
	PRIMARY KEY ("miner_id", "sector_id")
);

CREATE TABLE IF NOT EXISTS "miner_powers" (
	"miner_id" text NOT NULL,
	"state_root" text NOT NULL,
	"raw_byte_power" text NOT NULL,
	"quality_adjusted_power" text NOT NULL,
	PRIMARY KEY ("miner_id", "state_root")
);

CREATE TABLE IF NOT EXISTS "miner_pre_commit_infos" (
	"miner_id" text NOT NULL,
	"sector_id" bigserial,
	"state_root" text NOT NULL,
	"sealed_cid" text NOT NULL,
	"seal_rand_epoch" bigint,
	"expiration_epoch" bigint,
	"pre_commit_deposit" text NOT NULL,
	"pre_commit_epoch" bigint,
	"deal_weight" text NOT NULL,
	"verified_deal_weight" text NOT NULL,
	"is_replace_capacity" boolean,
	"replace_sector_deadline" bigint,
	"replace_sector_partition" bigint,
	"replace_sector_number" bigint,
	PRIMARY KEY ("miner_id", "sector_id", "state_root")
);

CREATE TABLE IF NOT EXISTS "miner_sector_infos" (
	"miner_id" text NOT NULL,
	"sector_id" bigserial,
	"state_root" text NOT NULL,
	"sealed_cid" text NOT NULL,
	"activation_epoch" bigint,
	"expiration_epoch" bigint,
	"deal_weight" text NOT NULL,
	"verified_deal_weight" text NOT NULL,
	"initial_pledge" text NOT NULL,
	"expected_day_reward" text NOT NULL,
	"expected_storage_pledge" text NOT NULL,
	PRIMARY KEY ("miner_id", "sector_id", "state_root")
);

CREATE TABLE IF NOT EXISTS "miner_states" (
	"miner_id" text NOT NULL,
	"owner_id" text NOT NULL,
	"worker_id" text NOT NULL,
	"peer_id" bytea,
	"sector_size" text NOT NULL,
	PRIMARY KEY ("miner_id")
);

ALTER TABLE public.receipts RENAME COLUMN msg TO message;
ALTER TABLE public.receipts RENAME COLUMN state TO state_root;
ALTER TABLE public.receipts RENAME COLUMN exit TO exit_code;
ALTER TABLE public.receipts ALTER idx TYPE bigint;
ALTER TABLE public.receipts ALTER exit_code TYPE bigint;

`)
	// A

	down := batch(`
ALTER TABLE public.actors RENAME COLUMN state_root TO stateroot;
ALTER TABLE public.actors ALTER nonce TYPE integer;

ALTER TABLE public.actor_states ALTER COLUMN state TYPE jsonb using state::JSON;

ALTER TABLE public.block_headers RENAME TO blocks;
ALTER TABLE public.blocks RENAME COLUMN fork_signaling TO forksig;
ALTER TABLE public.blocks RENAME COLUMN parent_state_root TO parentstateroot;
ALTER TABLE public.blocks RENAME COLUMN parent_weight TO parentweight;
ALTER TABLE public.blocks ALTER COLUMN parentweight TYPE integer USING (parentweight::integer);

ALTER TABLE public.blocks_synced DROP COLUMN height;
ALTER TABLE public.blocks_synced DROP COLUMN completed_at;

ALTER TABLE public.blocks_synced
ALTER synced_at TYPE int
USING extract(EPOCH from synced_at);

ALTER TABLE public.blocks_synced
ALTER processed_at TYPE int
USING extract(EPOCH from processed_at);

DROP TABLE IF EXISTS public.chain_powers;
DROP TABLE IF EXISTS public.chain_rewards;

ALTER TABLE public.drand_block_entries RENAME TO block_drand_entries;

DROP TABLE IF EXISTS public.id_addresses;

ALTER TABLE public.market_deal_proposals DROP COLUMN label;

-- See slashed_epoch TODO above
-- ALTER TABLE public.market_deal_proposals ADD COLUMN slashed_epoch bigint;

DROP TABLE IF EXISTS public.miner_deal_sectors;
DROP TABLE IF EXISTS public.miner_powers;
DROP TABLE IF EXISTS public.miner_pre_commit_infos;
DROP TABLE IF EXISTS public.miner_sector_infos;
DROP TABLE IF EXISTS public.miner_states;

ALTER TABLE public.receipts RENAME COLUMN message TO msg;
ALTER TABLE public.receipts RENAME COLUMN state_root TO state;
ALTER TABLE public.receipts RENAME COLUMN exit_code TO exit;
ALTER TABLE public.receipts ALTER idx TYPE integer;
ALTER TABLE public.receipts ALTER exit type integer;

`)

	patches.MustRegisterTx(up, down)
}
