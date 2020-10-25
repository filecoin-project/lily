package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 14 paritions most tables by height as timescale hypertables

func init() {
	up := batch(`

-- Ideally we want chunks to fit into memory so aim to keep them fairly small
-- Best practise is to restrict chunk sizes so total across all active hypertables fits 25% main memory
-- Space partitioning can also be used (i.e. hash over another column) but main benefit comes from
-- parallelising IO by placing those chunks on separate disks which we are not planning at the moment.
-- Space partitioning can be added at a later date via the add_dimension function.

-- TimescaleDB only supports UNIQUE constraints that have the partitioning key as their prefix. This
-- means that all out unique indexes have to include height so that timescale can place them in the
-- correct chunk.

-- ----------------------------------------------------------------
-- visor_processing_tipsets
-- ----------------------------------------------------------------

-- Make sure height is in the primary key of visor_processing_tipsets
ALTER TABLE visor_processing_tipsets ALTER COLUMN height SET NOT NULL;
ALTER TABLE visor_processing_tipsets DROP CONSTRAINT IF EXISTS visor_processing_statechanges_pkey;
ALTER TABLE visor_processing_tipsets ADD PRIMARY KEY (height, tip_set);

DROP INDEX IF EXISTS visor_processing_tipsets_statechange_idx;
DROP INDEX IF EXISTS visor_processing_tipsets_message_idx;
DROP INDEX IF EXISTS visor_processing_tipsets_economics_idx;
DROP INDEX IF EXISTS visor_processing_tipsets_height_idx;

-- Specific indexes for selection of work by tasks
CREATE INDEX IF NOT EXISTS "visor_processing_tipsets_message_idx" ON public.visor_processing_tipsets USING BTREE (height,message_claimed_until,message_completed_at);
CREATE INDEX IF NOT EXISTS "visor_processing_tipsets_statechange_idx" ON public.visor_processing_tipsets USING BTREE (height,statechange_claimed_until,statechange_completed_at);
CREATE INDEX IF NOT EXISTS "visor_processing_tipsets_economics_idx" ON public.visor_processing_tipsets USING BTREE (height,economics_claimed_until,economics_completed_at);

-- Convert visor_processing_tipsets to a hypertable partitioned on height (time)
-- Assume ~1103 bytes per row
-- Height chunked per week so we expect ~20160 rows per chunk, ~21MiB per chunk
SELECT create_hypertable(
	'visor_processing_tipsets',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);


-- ----------------------------------------------------------------
-- visor_processing_messages
-- ----------------------------------------------------------------

-- Make sure height is in the primary key of visor_processing_messages
ALTER TABLE visor_processing_messages ALTER COLUMN height SET NOT NULL;
ALTER TABLE visor_processing_messages DROP CONSTRAINT IF EXISTS visor_processing_messages_pkey;
ALTER TABLE visor_processing_messages ADD PRIMARY KEY (height, cid);

DROP INDEX IF EXISTS visor_processing_messages_gas_outputs_idx;
DROP INDEX IF EXISTS visor_processing_messages_height_idx;

-- Specific index for selection of work by tasks
CREATE INDEX IF NOT EXISTS "visor_processing_messages_gas_outputs_idx" ON public.visor_processing_messages USING BTREE (height,gas_outputs_claimed_until,gas_outputs_completed_at);

-- Convert visor_processing_messages to a hypertable partitioned on height (time)
-- Assume ~250 messages per epoch, ~280 bytes per row
-- Height chunked per day so we expect 8640*250 = ~700000 rows per chunk, ~187MiB per chunk
SELECT create_hypertable(
	'visor_processing_messages',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE
);


-- ----------------------------------------------------------------
-- visor_processing_actors
-- ----------------------------------------------------------------

-- Make sure height is in the primary key of visor_processing_actors
ALTER TABLE visor_processing_actors ALTER COLUMN height SET NOT NULL;
ALTER TABLE visor_processing_actors DROP CONSTRAINT IF EXISTS visor_processing_actors_pkey;
ALTER TABLE visor_processing_actors ADD PRIMARY KEY (height, head, code);

DROP INDEX IF EXISTS visor_processing_actors_completed_idx;
DROP INDEX IF EXISTS visor_processing_actors_code_idx;
DROP INDEX IF EXISTS visor_processing_actors_height_idx;

-- Specific indexes for selection of work by tasks
CREATE INDEX IF NOT EXISTS "visor_processing_actors_claimed_idx" ON public.visor_processing_actors USING BTREE (height,claimed_until,completed_at);
CREATE INDEX IF NOT EXISTS "visor_processing_actors_codeclaimed_idx" ON public.visor_processing_actors USING BTREE (code,height,claimed_until,completed_at);


-- Convert visor_processing_actors to a hypertable partitioned on height (time)
-- Assume ~20 state changes per epoch, ~1071 bytes per table row
-- Height chunked per 2 days so we expect 5760*20 = ~115200 rows per chunk, ~118MiB per chunk
SELECT create_hypertable(
	'visor_processing_actors',
	'height',
	chunk_time_interval => 5760,
	if_not_exists => TRUE
);



-- block_headers table used to be called blocks
ALTER TABLE block_headers DROP CONSTRAINT IF EXISTS blocks_pk;
ALTER TABLE block_headers ADD PRIMARY KEY (height, cid);
DROP INDEX IF EXISTS block_cid_uindex;
CREATE INDEX IF NOT EXISTS "block_headers_timestamp_idx" ON public.block_headers USING BTREE (timestamp);


-- Convert block_headers to a hypertable partitioned on height (time)
-- Assume ~5 blocks per epoch, ~432 bytes per table row
-- Height chunked per week so we expect 20160*5 = ~100800 rows per chunk, ~42MiB per chunk
SELECT create_hypertable(
	'block_headers',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);

-- this will fail if block_parents is populated since new height column is not null
ALTER TABLE public.block_parents ADD COLUMN height bigint NOT NULL;
DROP INDEX IF EXISTS block_parents_block_parent_uindex;
ALTER TABLE block_parents ADD PRIMARY KEY (height, block, parent);

-- Convert block_parents to a hypertable partitioned on height (time)
-- Assume ~5 blocks per epoch with ~4 parents, ~150 bytes per table row
-- Height chunked per week so we expect 20160*5*4 = ~403200 rows per chunk, ~58MiB per chunk
SELECT create_hypertable(
	'block_parents',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);


-- this will fail if block_messages is populated since new height column is not null
ALTER TABLE public.block_messages ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.block_messages DROP CONSTRAINT IF EXISTS block_messages_pk;
ALTER TABLE public.block_messages ADD PRIMARY KEY (height, block, message);

-- Convert block_messages to a hypertable partitioned on height (time)
-- Assume ~250 messages per epoch, ~200 bytes per table row
-- Height chunked per day so we expect 2880*250 = ~720000 rows per chunk, ~137MiB per chunk
SELECT create_hypertable(
	'block_messages',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE
);

-- this will fail if messages is populated since new height column is not null
ALTER TABLE public.messages ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.messages DROP CONSTRAINT IF EXISTS messages_pk;
ALTER TABLE public.messages ADD PRIMARY KEY (height, cid);
DROP INDEX IF EXISTS messages_cid_uindex;

-- Convert messages to a hypertable partitioned on height (time)
-- Assume ~250 messages per epoch, ~373 bytes per table row (not including toast)
-- Height chunked per day so we expect 2880*250 = ~720000 rows per chunk, ~256MiB per chunk
SELECT create_hypertable(
	'messages',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE
);

ALTER TABLE public.parsed_messages DROP CONSTRAINT IF EXISTS parsed_messages_pkey;
ALTER TABLE public.parsed_messages ADD PRIMARY KEY (height, cid);

-- Convert messages to a hypertable partitioned on height (time)
-- Assume ~250 messages per epoch, ~373 bytes per table row (not including toast for jsonb)
-- Height chunked per day so we expect 2880*250 = ~720000 rows per chunk, ~256MiB per chunk
SELECT create_hypertable(
	'parsed_messages',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE
);


-- this will fail if receipts is populated since new height column is not null
ALTER TABLE public.receipts ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.receipts DROP CONSTRAINT IF EXISTS receipts_pk;
ALTER TABLE public.receipts ADD PRIMARY KEY (height, message, state_root);
DROP INDEX IF EXISTS receipts_msg_state_index;

-- Convert receipts to a hypertable partitioned on height (time)
-- Assume ~250 receipts per epoch, ~215 bytes per table row
-- Height chunked per day so we expect 2880*250 = ~720000 rows per chunk, ~148MiB per chunk
SELECT create_hypertable(
	'receipts',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE
);


-- this will fail if message_gas_economy is populated since new height column is not null
ALTER TABLE public.message_gas_economy ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.message_gas_economy DROP CONSTRAINT IF EXISTS message_gas_economy_pkey;
ALTER TABLE public.message_gas_economy ADD PRIMARY KEY (height, state_root);

-- Convert message_gas_economy to a hypertable partitioned on height (time)
-- Assume ~1 row per epoch, ~800 bytes per table row
-- Height chunked per week so we expect 20160*800 = ~15MiB per chunk
SELECT create_hypertable(
	'message_gas_economy',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);


-- ----------------------------------------------------------------
-- actors
-- ----------------------------------------------------------------

ALTER TABLE public.actors ADD COLUMN height bigint NOT NULL;
DROP INDEX IF EXISTS actors_id_index;
ALTER TABLE public.actors ADD PRIMARY KEY (height, id, state_root);

-- Convert actors to a hypertable partitioned on height (time)
-- Assume ~20 state changes per epoch, ~250 bytes per table row
-- Height chunked per 7 days so we expect 20160*20 = ~403200 rows per chunk, ~96MiB per chunk
SELECT create_hypertable(
	'actors',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);

-- ----------------------------------------------------------------
-- actor_states
-- ----------------------------------------------------------------

ALTER TABLE public.actor_states ADD COLUMN height bigint NOT NULL;
DROP INDEX IF EXISTS actor_states_head_code_uindex;
DROP INDEX IF EXISTS actor_states_code_head_index;
DROP INDEX IF EXISTS actor_states_head_index;
ALTER TABLE public.actor_states ADD PRIMARY KEY (height, head, code);

-- Convert actor_states to a hypertable partitioned on height (time)
-- Assume ~20 state changes per epoch, ~850 bytes per table row
-- Height chunked per 4 days so we expect 11520*20 = ~230400 rows per chunk, ~186MiB per chunk
SELECT create_hypertable(
	'actor_states',
	'height',
	chunk_time_interval => 11520,
	if_not_exists => TRUE
);

-- ----------------------------------------------------------------
-- chain_rewards
-- ----------------------------------------------------------------

ALTER TABLE public.chain_rewards ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.chain_rewards DROP CONSTRAINT IF EXISTS chain_rewards_pkey;
ALTER TABLE public.chain_rewards ADD PRIMARY KEY (height, state_root);

-- Convert chain_rewards to a hypertable partitioned on height (time)
-- Assume ~1  per epoch, ~400 bytes per table row
-- Height chunked per 28 days so we expect ~80640 rows per chunk, ~28MiB per chunk
SELECT create_hypertable(
	'chain_rewards',
	'height',
	chunk_time_interval => 80640,
	if_not_exists => TRUE
);

-- ----------------------------------------------------------------
-- chain_powers
-- ----------------------------------------------------------------

ALTER TABLE public.chain_powers ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.chain_powers DROP CONSTRAINT IF EXISTS chain_powers_pkey;
ALTER TABLE public.chain_powers ADD PRIMARY KEY (height, state_root);

-- Convert chain_powers to a hypertable partitioned on height (time)
-- Assume ~1  per epoch, ~400 bytes per table row
-- Height chunked per 28 days so we expect ~80640 rows per chunk, ~28MiB per chunk
SELECT create_hypertable(
	'chain_powers',
	'height',
	chunk_time_interval => 80640,
	if_not_exists => TRUE
);

-- ----------------------------------------------------------------
-- miner_powers
-- ----------------------------------------------------------------

ALTER TABLE public.miner_powers ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.miner_powers DROP CONSTRAINT IF EXISTS miner_powers_pkey;
ALTER TABLE public.miner_powers ADD PRIMARY KEY (height, miner_id, state_root);

-- Convert miner_powers to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~150 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~14MiB per chunk
SELECT create_hypertable(
	'miner_powers',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);


-- ----------------------------------------------------------------
-- miner_pre_commit_infos
-- ----------------------------------------------------------------

-- sector_id was an autoincrement serial type by mistake
ALTER TABLE public.miner_pre_commit_infos ALTER COLUMN sector_id DROP DEFAULT;
DROP SEQUENCE IF EXISTS miner_pre_commit_infos_sector_id_seq;


ALTER TABLE public.miner_pre_commit_infos ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.miner_pre_commit_infos DROP CONSTRAINT IF EXISTS miner_pre_commit_infos_pkey;
ALTER TABLE public.miner_pre_commit_infos ADD PRIMARY KEY (height, miner_id, sector_id, state_root);

-- Convert miner_pre_commit_infos to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~300 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~28MiB per chunk
SELECT create_hypertable(
	'miner_pre_commit_infos',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);


-- ----------------------------------------------------------------
-- miner_sector_infos
-- ----------------------------------------------------------------

-- sector_id was an autoincrement serial type by mistake
ALTER TABLE public.miner_sector_infos ALTER COLUMN sector_id DROP DEFAULT;
DROP SEQUENCE IF EXISTS miner_sector_infos_sector_id_seq;

ALTER TABLE public.miner_sector_infos ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.miner_sector_infos DROP CONSTRAINT IF EXISTS miner_sector_infos_pkey;
ALTER TABLE public.miner_sector_infos ADD PRIMARY KEY (height, miner_id, sector_id, state_root);

-- Convert miner_sector_infos to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~300 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~28MiB per chunk
SELECT create_hypertable(
	'miner_sector_infos',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);

-- ----------------------------------------------------------------
-- miner_sector_events
-- ----------------------------------------------------------------

ALTER TABLE public.miner_sector_events ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.miner_sector_events DROP CONSTRAINT IF EXISTS miner_sector_events_pk;
ALTER TABLE public.miner_sector_events ADD PRIMARY KEY (height, sector_id, event, miner_id, state_root);

-- Convert miner_sector_events to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~300 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~28MiB per chunk
SELECT create_hypertable(
	'miner_sector_events',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);

-- ----------------------------------------------------------------
-- miner_deal_sectors
-- ----------------------------------------------------------------

-- sector_id was an autoincrement serial type by mistake
ALTER TABLE public.miner_deal_sectors ALTER COLUMN sector_id DROP DEFAULT;
DROP SEQUENCE IF EXISTS miner_deal_sectors_sector_id_seq;

ALTER TABLE public.miner_deal_sectors ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.miner_deal_sectors DROP CONSTRAINT IF EXISTS miner_deal_sectors_pkey;
ALTER TABLE public.miner_deal_sectors ADD PRIMARY KEY (height, miner_id, sector_id);

-- Convert miner_deal_sectors to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~150 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~14MiB per chunk
SELECT create_hypertable(
	'miner_deal_sectors',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);

-- ----------------------------------------------------------------
-- miner_sector_posts
-- ----------------------------------------------------------------

ALTER TABLE public.miner_sector_posts RENAME COLUMN epoch TO height;
ALTER TABLE public.miner_sector_posts DROP CONSTRAINT IF EXISTS miner_sector_posts_pkey;
ALTER TABLE public.miner_sector_posts ADD PRIMARY KEY (height, miner_id, sector_id);

-- Convert miner_sector_posts to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~150 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~14MiB per chunk
SELECT create_hypertable(
	'miner_sector_posts',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);

-- ----------------------------------------------------------------
-- miner_states
-- ----------------------------------------------------------------

ALTER TABLE public.miner_states ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.miner_states DROP CONSTRAINT IF EXISTS miner_states_pkey;
ALTER TABLE public.miner_states ADD PRIMARY KEY (height, miner_id);

-- Convert miner_states to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~150 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~14MiB per chunk
SELECT create_hypertable(
	'miner_states',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);


-- ----------------------------------------------------------------
-- market_deal_states
-- ----------------------------------------------------------------

ALTER TABLE public.market_deal_states ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.market_deal_states DROP CONSTRAINT IF EXISTS market_deal_states_pk;
ALTER TABLE public.market_deal_states ADD PRIMARY KEY (height, deal_id, state_root);
ALTER TABLE public.market_deal_states DROP CONSTRAINT IF EXISTS market_deal_states_deal_id_sector_start_epoch_last_update_e_key;

-- Convert market_deal_states to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~150 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~14MiB per chunk
SELECT create_hypertable(
	'market_deal_states',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);


-- ----------------------------------------------------------------
-- market_deal_proposals
-- ----------------------------------------------------------------

ALTER TABLE public.market_deal_proposals ADD COLUMN height bigint NOT NULL;
ALTER TABLE public.market_deal_proposals DROP CONSTRAINT IF EXISTS market_deal_proposal_pk;
ALTER TABLE public.market_deal_proposals ADD PRIMARY KEY (height, deal_id);

-- Convert market_deal_proposals to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~350 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, 34MiB per chunk
SELECT create_hypertable(
	'market_deal_proposals',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);


`)
	// there is no simple way to convert a hypertable back to a normal table
	down := batch(`SELECT 1`)

	migrations.MustRegisterTx(up, down)
}
