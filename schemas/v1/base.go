package v1

// BaseTemplate is the template the initial schema for this major version. The template expects variables to be
// passed using the schema.Config struct. Patches are applied on top of this base.
var BaseTemplate = `

{{- if and .SchemaName (ne .SchemaName "public") }}
SET search_path TO {{ .SchemaName }},public;
{{- end }}

-- =====================================================================================================================
-- TYPES
-- =====================================================================================================================

CREATE TYPE {{ .SchemaName | default "public"}}.miner_sector_event_type AS ENUM (
    'PRECOMMIT_ADDED',
    'PRECOMMIT_EXPIRED',
    'COMMIT_CAPACITY_ADDED',
    'SECTOR_ADDED',
    'SECTOR_EXTENDED',
    'SECTOR_EXPIRED',
    'SECTOR_FAULTED',
    'SECTOR_RECOVERING',
    'SECTOR_RECOVERED',
    'SECTOR_TERMINATED'
);

-- =====================================================================================================================
-- INDEPENDENT FUNCTIONS
-- =====================================================================================================================

CREATE FUNCTION {{ .SchemaName | default "public"}}.height_to_unix(fil_epoch bigint) RETURNS bigint
    LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
    AS $$
		SELECT ((fil_epoch * 30) + 1598306400)::bigint;
	$$;

CREATE FUNCTION {{ .SchemaName | default "public"}}.unix_to_height(unix_epoch bigint) RETURNS bigint
    LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
    AS $$
		SELECT ((unix_epoch - 1598306400) / 30)::bigint;
	$$;

-- Note: system function 'now' is STABLE PARALLEL SAFE STRICT
CREATE FUNCTION {{ .SchemaName | default "public"}}.current_height() RETURNS bigint
    LANGUAGE sql STABLE PARALLEL SAFE STRICT
	AS $$
		SELECT unix_to_height(extract(epoch from now() AT TIME ZONE 'UTC')::bigint);
	$$;


-- =====================================================================================================================
-- TABLES
-- =====================================================================================================================

-- ----------------------------------------------------------------
-- Name: actor_states
-- Model: common.ActorState
-- Growth: About 650 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.actor_states (
    head text NOT NULL,
    code text NOT NULL,
    state jsonb NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.actor_states ADD CONSTRAINT actor_states_pkey PRIMARY KEY (height, head, code);
CREATE INDEX actor_states_height_idx ON {{ .SchemaName | default "public"}}.actor_states USING btree (height DESC);

-- Convert actor_states to a hypertable partitioned on height (time)
-- Assume ~20 state changes per epoch, ~850 bytes per table row
-- Height chunked per 4 days so we expect 11520*650 = ~7488000 rows per chunk, ~4.6GiB per chunk
SELECT create_hypertable(
	'actor_states',
	'height',
	chunk_time_interval => 11520,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('actor_states', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.actor_states IS 'Actor states that were changed at an epoch. Associates actors states as single-level trees with CIDs pointing to complete state tree with the root CID (head) for that actor''s state.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_states.head IS 'CID of the root of the state tree for the actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_states.code IS 'CID identifier for the type of the actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_states.state IS 'Top level of state data.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_states.height IS 'Epoch when this state change happened.';


-- ----------------------------------------------------------------
-- Name: actors
-- Model: common.Actor
-- Growth: About 1300 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.actors (
    id text NOT NULL,
    code text NOT NULL,
    head text NOT NULL,
    nonce bigint NOT NULL,
    balance text NOT NULL,
    state_root text NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.actors ADD CONSTRAINT actors_pkey PRIMARY KEY (height, id, state_root);
CREATE INDEX actors_height_idx ON {{ .SchemaName | default "public"}}.actors USING btree (height DESC);

-- Convert actors to a hypertable partitioned on height (time)
-- Assume ~20 state changes per epoch, ~250 bytes per table row
-- Height chunked per 7 days so we expect 20160*1300 = ~26208000 rows per chunk, ~6.2GiB per chunk
SELECT create_hypertable(
	'actors',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('actors', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.actors IS 'Actors on chain that were added or updated at an epoch. Associates the actor''s state root CID (head) with the chain state root CID from which it decends. Includes account ID nonce and balance at each state.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.id IS 'Actor address.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.code IS 'Human readable identifier for the type of the actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.head IS 'CID of the root of the state tree for the actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.nonce IS 'The next actor nonce that is expected to appear on chain.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.balance IS 'Actor balance in attoFIL.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.state_root IS 'CID of the state root.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.height IS 'Epoch when this actor was created or updated.';


-- ----------------------------------------------------------------
-- Name: blocks.block_headers
-- Model: blocks.BlockHeader
-- Growth: About 4-5 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.block_headers (
    cid text NOT NULL,
    parent_weight text NOT NULL,
    parent_state_root text NOT NULL,
    height bigint NOT NULL,
    miner text NOT NULL,
    "timestamp" bigint NOT NULL,
    win_count bigint,
    parent_base_fee text NOT NULL,
    fork_signaling bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.block_headers ADD CONSTRAINT block_headers_pkey PRIMARY KEY (height, cid);
CREATE INDEX block_headers_height_idx ON {{ .SchemaName | default "public"}}.block_headers USING btree (height DESC);
CREATE INDEX block_headers_timestamp_idx ON {{ .SchemaName | default "public"}}.block_headers USING btree ("timestamp");

-- Convert block_headers to a hypertable partitioned on height (time)
-- Assume ~5 blocks per epoch, ~432 bytes per table row
-- Height chunked per week so we expect 20160*5 = ~100800 rows per chunk, ~42MiB per chunk
SELECT create_hypertable(
	'block_headers',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('block_headers', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.block_headers IS 'Blocks included in tipsets at an epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers.cid IS 'CID of the block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers.parent_weight IS 'Aggregate chain weight of the block''s parent set.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers.parent_state_root IS 'CID of the block''s parent state root.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers.height IS 'Epoch when this block was mined.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers.miner IS 'Address of the miner who mined this block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers."timestamp" IS 'Time the block was mined in Unix time, the number of seconds elapsed since January 1, 1970 UTC.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers.win_count IS 'Number of reward units won in this block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers.parent_base_fee IS 'The base fee after executing the parent tipset.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_headers.fork_signaling IS 'Flag used as part of signaling forks.';


-- ----------------------------------------------------------------
-- Name: block_messages
-- Model: messages.BlockMessage
-- Growth: About 900 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.block_messages (
    block text NOT NULL,
    message text NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.block_messages ADD CONSTRAINT block_messages_pkey PRIMARY KEY (height, block, message);
CREATE INDEX block_messages_height_idx ON {{ .SchemaName | default "public"}}.block_messages USING btree (height DESC);

-- Convert block_messages to a hypertable partitioned on height (time)
-- Assume ~250 messages per epoch, ~200 bytes per table row
-- Height chunked per day so we expect 2880*900 = ~2592000 rows per chunk, ~500MiB per chunk
SELECT create_hypertable(
	'block_messages',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('block_messages', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.block_messages IS 'Message CIDs and the Blocks CID which contain them.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_messages.block IS 'CID of the block that contains the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_messages.message IS 'CID of a message in the block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_messages.height IS 'Epoch when the block was mined.';


-- ----------------------------------------------------------------
-- Name: block_parents
-- Model: blocks.BlockParent
-- Growth: About 20 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.block_parents (
    block text NOT NULL,
    parent text NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.block_parents ADD CONSTRAINT block_parents_pkey PRIMARY KEY (height, block, parent);
CREATE INDEX block_parents_height_idx ON {{ .SchemaName | default "public"}}.block_parents USING btree (height DESC);

-- Convert block_parents to a hypertable partitioned on height (time)
-- Assume ~5 blocks per epoch with ~4 parents, ~150 bytes per table row
-- Height chunked per week so we expect 20160*5*4 = ~403200 rows per chunk, ~58MiB per chunk
SELECT create_hypertable(
	'block_parents',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('block_parents', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.block_parents IS 'Block CIDs to many parent Block CIDs.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_parents.block IS 'CID of the block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_parents.parent IS 'CID of the parent block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.block_parents.height IS 'Epoch when the block was mined.';

-- ----------------------------------------------------------------
-- Name: chain_economics
-- Model: chain.ChainEconomics
-- Growth: One row per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.chain_economics (
    height bigint NOT NULL,
    parent_state_root text NOT NULL,
    circulating_fil numeric NOT NULL,
    vested_fil numeric NOT NULL,
    mined_fil numeric NOT NULL,
    burnt_fil numeric NOT NULL,
    locked_fil numeric NOT NULL,
    fil_reserve_disbursed numeric NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.chain_economics ADD CONSTRAINT chain_economics_pk PRIMARY KEY (height, parent_state_root);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.chain_economics IS 'Economic summaries per state root CID.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_economics.height IS 'Epoch of the economic summary.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_economics.parent_state_root IS 'CID of the parent state root.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_economics.circulating_fil IS 'The amount of FIL (attoFIL) circulating and tradeable in the economy. The basis for Market Cap calculations.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_economics.vested_fil IS 'Total amount of FIL (attoFIL) that is vested from genesis allocation.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_economics.mined_fil IS 'The amount of FIL (attoFIL) that has been mined by storage miners.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_economics.burnt_fil IS 'Total FIL (attoFIL) burned as part of penalties and on-chain computations.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_economics.locked_fil IS 'The amount of FIL (attoFIL) locked as part of mining, deals, and other mechanisms.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_economics.fil_reserve_disbursed IS 'The amount of FIL (attoFIL) that has been disbursed from the mining reserve.';


-- ----------------------------------------------------------------
-- Name: chain_powers
-- Model: chain.ChainPower
-- Growth: One row per epoch
-- Notes: This was a hypertable in v0, removed since it only grows 1 row per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.chain_powers (
    state_root text NOT NULL,
    total_raw_bytes_power numeric NOT NULL,
    total_raw_bytes_committed numeric NOT NULL,
    total_qa_bytes_power numeric NOT NULL,
    total_qa_bytes_committed numeric NOT NULL,
    total_pledge_collateral numeric NOT NULL,
    qa_smoothed_position_estimate numeric NOT NULL,
    qa_smoothed_velocity_estimate numeric NOT NULL,
    miner_count bigint,
    participating_miner_count bigint,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.chain_powers ADD CONSTRAINT chain_powers_pkey PRIMARY KEY (height, state_root);
CREATE INDEX chain_powers_height_idx ON {{ .SchemaName | default "public"}}.chain_powers USING btree (height DESC);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.chain_powers IS 'Power summaries from the Power actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.state_root IS 'CID of the parent state root.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.total_raw_bytes_power IS 'Total storage power in bytes in the network. Raw byte power is the size of a sector in bytes.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.total_raw_bytes_committed IS 'Total provably committed storage power in bytes. Raw byte power is the size of a sector in bytes.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.total_qa_bytes_power IS 'Total quality adjusted storage power in bytes in the network. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.total_qa_bytes_committed IS 'Total provably committed, quality adjusted storage power in bytes. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.total_pledge_collateral IS 'Total locked FIL (attoFIL) miners have pledged as collateral in order to participate in the economy.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.qa_smoothed_position_estimate IS 'Total power smoothed position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.qa_smoothed_velocity_estimate IS 'Total power smoothed velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.miner_count IS 'Total number of miners.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.participating_miner_count IS 'Total number of miners with power above the minimum miner threshold.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_powers.height IS 'Epoch this power summary applies to.';


-- ----------------------------------------------------------------
-- Name: chain_rewards
-- Model: reward.ChainReward
-- Growth: One row per epoch
-- Notes: This was a hypertable in v0, removed since it only grows 1 row per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.chain_rewards (
    state_root text NOT NULL,
    cum_sum_baseline numeric NOT NULL,
    cum_sum_realized numeric NOT NULL,
    effective_baseline_power numeric NOT NULL,
    new_baseline_power numeric NOT NULL,
    new_reward_smoothed_position_estimate numeric NOT NULL,
    new_reward_smoothed_velocity_estimate numeric NOT NULL,
    total_mined_reward numeric NOT NULL,
    new_reward numeric,
    effective_network_time bigint,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.chain_rewards ADD CONSTRAINT chain_rewards_pkey PRIMARY KEY (height, state_root);
CREATE INDEX chain_rewards_height_idx ON {{ .SchemaName | default "public"}}.chain_rewards USING btree (height DESC);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.chain_rewards IS 'Reward summaries from the Reward actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.state_root IS 'CID of the parent state root.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.cum_sum_baseline IS 'Target that CumsumRealized needs to reach for EffectiveNetworkTime to increase. It is measured in byte-epochs (space * time) representing power committed to the network for some duration.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.cum_sum_realized IS 'Cumulative sum of network power capped by BaselinePower(epoch). It is measured in byte-epochs (space * time) representing power committed to the network for some duration.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.effective_baseline_power IS 'The baseline power (in bytes) at the EffectiveNetworkTime epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.new_baseline_power IS 'The baseline power (in bytes) the network is targeting.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.new_reward_smoothed_position_estimate IS 'Smoothed reward position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.new_reward_smoothed_velocity_estimate IS 'Smoothed reward velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.total_mined_reward IS 'The total FIL (attoFIL) awarded to block miners.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.new_reward IS 'The reward to be paid in per WinCount to block producers. The actual reward total paid out depends on the number of winners in any round. This value is recomputed every non-null epoch and used in the next non-null epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.effective_network_time IS 'Ceiling of real effective network time "theta" based on CumsumBaselinePower(theta) == CumsumRealizedPower. Theta captures the notion of how much the network has progressed in its baseline and in advancing network time.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_rewards.height IS 'Epoch this rewards summary applies to.';

-- ----------------------------------------------------------------
-- Name: derived_gas_outputs
-- Model: derived.GasOutputs
-- Growth: About 340 rows per epoch
-- Notes: Converted to hypertable
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.derived_gas_outputs (
    cid text NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    value numeric NOT NULL,
    gas_fee_cap numeric NOT NULL,
    gas_premium numeric NOT NULL,
    gas_limit bigint,
    size_bytes bigint,
    nonce bigint,
    method bigint,
    state_root text NOT NULL,
    exit_code bigint NOT NULL,
    gas_used bigint NOT NULL,
    parent_base_fee numeric NOT NULL,
    base_fee_burn numeric NOT NULL,
    over_estimation_burn numeric NOT NULL,
    miner_penalty numeric NOT NULL,
    miner_tip numeric NOT NULL,
    refund numeric NOT NULL,
    gas_refund bigint NOT NULL,
    gas_burned bigint NOT NULL,
    height bigint NOT NULL,
    actor_name text NOT NULL,
    actor_family text NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.derived_gas_outputs ADD CONSTRAINT derived_gas_outputs_pkey PRIMARY KEY (height, cid, state_root);
CREATE INDEX derived_gas_outputs_exit_code_index ON {{ .SchemaName | default "public"}}.derived_gas_outputs USING btree (exit_code);
CREATE INDEX derived_gas_outputs_from_index ON {{ .SchemaName | default "public"}}.derived_gas_outputs USING hash ("from");
CREATE INDEX derived_gas_outputs_method_index ON {{ .SchemaName | default "public"}}.derived_gas_outputs USING btree (method);
CREATE INDEX derived_gas_outputs_to_index ON {{ .SchemaName | default "public"}}.derived_gas_outputs USING hash ("to");
CREATE INDEX derived_gas_outputs_actor_family_index ON {{ .SchemaName | default "public"}}.derived_gas_outputs USING btree ("actor_family");

-- Convert block_headers to a hypertable partitioned on height (time)
-- Assume ~340 rows per epoch, ~491 bytes per table row
-- Height chunked per week so we expect 20160*340 = ~6854400 rows per chunk, ~3.2GiB per chunk
SELECT create_hypertable(
	'derived_gas_outputs',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('derived_gas_outputs', 'current_height', replace_if_exists => true);


COMMENT ON TABLE {{ .SchemaName | default "public"}}.derived_gas_outputs IS 'Derived gas costs resulting from execution of a message in the VM.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.cid IS 'CID of the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs."from" IS 'Address of actor that sent the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs."to" IS 'Address of actor that received the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.value IS 'The FIL value transferred (attoFIL) to the message receiver.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.gas_fee_cap IS 'The maximum price that the message sender is willing to pay per unit of gas.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.gas_premium IS 'The price per unit of gas (measured in attoFIL/gas) that the message sender is willing to pay (on top of the BaseFee) to "tip" the miner that will include this message in a block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.gas_limit IS 'A hard limit on the amount of gas (i.e., number of units of gas) that a message’s execution should be allowed to consume on chain. It is measured in units of gas.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.size_bytes IS 'Size in bytes of the serialized message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.nonce IS 'The message nonce, which protects against duplicate messages and multiple messages with the same values.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.method IS 'The method number to invoke. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.state_root IS 'CID of the parent state root.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.exit_code IS 'The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.gas_used IS 'A measure of the amount of resources (or units of gas) consumed, in order to execute a message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.parent_base_fee IS 'The set price per unit of gas (measured in attoFIL/gas unit) to be burned (sent to an unrecoverable address) for every message execution.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.base_fee_burn IS 'The amount of FIL (in attoFIL) to burn as a result of the base fee. It is parent_base_fee (or gas_fee_cap if smaller) multiplied by gas_used. Note: successful window PoSt messages are not charged this burn.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.over_estimation_burn IS 'The fee to pay (in attoFIL) for overestimating the gas used to execute a message. The overestimated gas to burn (gas_burned) is a portion of the difference between gas_limit and gas_used. The over_estimation_burn value is gas_burned * parent_base_fee.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.miner_penalty IS 'Any penalty fees (in attoFIL) the miner incured while executing the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.miner_tip IS 'The amount of FIL (in attoFIL) the miner receives for executing the message. Typically it is gas_premium * gas_limit but may be lower if the total fees exceed the gas_fee_cap.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.refund IS 'The amount of FIL (in attoFIL) to refund to the message sender after base fee, miner tip and overestimation amounts have been deducted.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.gas_refund IS 'The overestimated units of gas to refund. It is a portion of the difference between gas_limit and gas_used.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.gas_burned IS 'The overestimated units of gas to burn. It is a portion of the difference between gas_limit and gas_used.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.height IS 'Epoch this message was executed at.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.derived_gas_outputs.actor_name IS 'Human readable identifier for the type of the actor.';


-- ----------------------------------------------------------------
-- Name: drand_block_entries
-- Model: blocks.DrandBlockEntrie
-- Growth: About 4 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.drand_block_entries (
    round bigint NOT NULL,
    block text NOT NULL
);
CREATE UNIQUE INDEX block_drand_entries_round_uindex ON {{ .SchemaName | default "public"}}.drand_block_entries USING btree (round, block);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.drand_block_entries IS 'Drand randomness round numbers used in each block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.drand_block_entries.round IS 'The round number of the randomness used.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.drand_block_entries.block IS 'CID of the block.';

-- ----------------------------------------------------------------
-- Name: gopg_migrations
-- Notes: This table and sequence can be created during version checking before a migration.
-- ----------------------------------------------------------------
ALTER SEQUENCE {{ .SchemaName | default "public"}}.gopg_migrations_id_seq OWNED BY {{ .SchemaName | default "public"}}.gopg_migrations.id;

CREATE SEQUENCE IF NOT EXISTS {{ .SchemaName | default "public"}}.gopg_migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.gopg_migrations (
    id integer NOT NULL,
    version bigint,
    created_at timestamp with time zone
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.gopg_migrations ALTER COLUMN id SET DEFAULT nextval('{{ .SchemaName | default "public"}}.gopg_migrations_id_seq'::regclass);


-- ----------------------------------------------------------------
-- Name: id_addresses
-- Model: init.IdAddress
-- Growth: About 1 row per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.id_addresses (
    height bigint NOT NULL,
    id text NOT NULL,
    address text NOT NULL,
    state_root text NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.id_addresses ADD CONSTRAINT id_addresses_pkey PRIMARY KEY (height, id, address, state_root);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.id_addresses IS 'Mapping of IDs to robust addresses from the init actor''s state.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.id_addresses.height IS 'Epoch at which this address mapping was added.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.id_addresses.id IS 'ID of the actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.id_addresses.address IS 'Robust address of the actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.id_addresses.state_root IS 'CID of the parent state root at which this address mapping was added.';

-- ----------------------------------------------------------------
-- Name: internal_messages
-- Model: messages.InternalMessage
-- Growth: Estimate ~400 per epoch, roughly same as messages
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.internal_messages (
    height bigint NOT NULL,
    cid text NOT NULL,
    state_root text NOT NULL,
    source_message text,
    "from" text NOT NULL,
    "to" text NOT NULL,
    value numeric NOT NULL,
    method bigint NOT NULL,
    actor_name text NOT NULL,
    actor_family text NOT NULL,
    exit_code bigint NOT NULL,
    gas_used bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.internal_messages ADD CONSTRAINT internal_messages_pkey PRIMARY KEY (height, cid);
CREATE INDEX internal_messages_exit_code_index ON {{ .SchemaName | default "public"}}.internal_messages USING btree (exit_code);
CREATE INDEX internal_messages_from_index ON {{ .SchemaName | default "public"}}.internal_messages USING hash ("from");
CREATE INDEX internal_messages_method_index ON {{ .SchemaName | default "public"}}.internal_messages USING btree (method);
CREATE INDEX internal_messages_to_index ON {{ .SchemaName | default "public"}}.internal_messages USING hash ("to");
CREATE INDEX internal_messages_actor_family_index ON {{ .SchemaName | default "public"}}.internal_messages USING btree ("actor_family");

-- Convert messages to a hypertable partitioned on height (time)
-- Height chunked per week so we expect 20160*400 = ~8064000 rows per chunk, ~2.8GiB per chunk
SELECT create_hypertable(
	'internal_messages',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('internal_messages', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.internal_messages IS 'Messages generated implicitly by system actors and by using the runtime send method.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.height IS 'Epoch this message was executed at.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.cid IS 'CID of the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.state_root IS 'CID of the parent state root at which this message was executed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.source_message IS 'CID of the message that caused this message to be sent.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages."from" IS 'Address of the actor that sent the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages."to" IS 'Address of the actor that received the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.value IS 'Amount of FIL (in attoFIL) transferred by this message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.method IS 'The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.actor_name IS 'The full versioned name of the actor that received the message (for example fil/3/storagepower).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.actor_family IS 'The short unversioned name of the actor that received the message (for example storagepower).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.exit_code IS 'The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_messages.gas_used IS 'A measure of the amount of resources (or units of gas) consumed, in order to execute a message.';

-- ----------------------------------------------------------------
-- Name: internal_parsed_messages
-- Model: messages.InternalParsedMessage
-- Growth: Estimate ~400 per epoch, roughly same as internal_messages
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.internal_parsed_messages (
    height bigint NOT NULL,
    cid text NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    value numeric NOT NULL,
    method text NOT NULL,
    params jsonb
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.internal_parsed_messages ADD CONSTRAINT internal_parsed_messages_pkey PRIMARY KEY (height, cid);
CREATE INDEX internal_parsed_messages_from_idx ON {{ .SchemaName | default "public"}}.internal_parsed_messages USING hash ("from");
CREATE INDEX internal_parsed_messages_method_idx ON {{ .SchemaName | default "public"}}.internal_parsed_messages USING hash (method);
CREATE INDEX internal_parsed_messages_to_idx ON {{ .SchemaName | default "public"}}.internal_parsed_messages USING hash ("to");

-- Convert messages to a hypertable partitioned on height (time)
-- Height chunked per week so we expect 20160*400 = ~8064000 rows per chunk, ~2.8GiB per chunk
SELECT create_hypertable(
	'internal_parsed_messages',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('internal_parsed_messages', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.internal_parsed_messages IS 'Internal messages parsed to extract useful information.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_parsed_messages.height IS 'Epoch this message was executed at.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_parsed_messages.cid IS 'CID of the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_parsed_messages."from" IS 'Address of the actor that sent the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_parsed_messages."to" IS 'Address of the actor that received the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_parsed_messages.value IS 'Amount of FIL (in attoFIL) transferred by this message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_parsed_messages.method IS 'The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.internal_parsed_messages.params IS 'Method parameters parsed and serialized as a JSON object.';


-- ----------------------------------------------------------------
-- Name: market_deal_proposals
-- Model: market.MarketDealProposal
-- Growth: About 2 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.market_deal_proposals (
    deal_id bigint NOT NULL,
    state_root text NOT NULL,
    piece_cid text NOT NULL,
    padded_piece_size bigint NOT NULL,
    unpadded_piece_size bigint NOT NULL,
    is_verified boolean NOT NULL,
    client_id text NOT NULL,
    provider_id text NOT NULL,
    start_epoch bigint NOT NULL,
    end_epoch bigint NOT NULL,
    slashed_epoch bigint,
    storage_price_per_epoch text NOT NULL,
    provider_collateral text NOT NULL,
    client_collateral text NOT NULL,
    label text,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.market_deal_proposals ADD CONSTRAINT market_deal_proposals_pkey PRIMARY KEY (height, deal_id);
CREATE INDEX market_deal_proposals_height_idx ON {{ .SchemaName | default "public"}}.market_deal_proposals USING btree (height DESC);

-- Convert market_deal_proposals to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~350 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, 34MiB per chunk
SELECT create_hypertable(
	'market_deal_proposals',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('market_deal_proposals', 'current_height', replace_if_exists => true);


COMMENT ON TABLE {{ .SchemaName | default "public"}}.market_deal_proposals IS 'All storage deal states with latest values applied to end_epoch when updates are detected on-chain.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.deal_id IS 'Identifier for the deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.state_root IS 'CID of the parent state root for this deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.piece_cid IS 'CID of a sector piece. A Piece is an object that represents a whole or part of a File.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.padded_piece_size IS 'The piece size in bytes with padding.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.unpadded_piece_size IS 'The piece size in bytes without padding.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.is_verified IS 'Deal is with a verified provider.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.client_id IS 'Address of the actor proposing the deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.provider_id IS 'Address of the actor providing the services.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.start_epoch IS 'The epoch at which this deal with begin. Storage deal must appear in a sealed (proven) sector no later than start_epoch, otherwise it is invalid.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.end_epoch IS 'The epoch at which this deal with end.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.storage_price_per_epoch IS 'The amount of FIL (in attoFIL) that will be transferred from the client to the provider every epoch this deal is active for.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.provider_collateral IS 'The amount of FIL (in attoFIL) the provider has pledged as collateral. The Client deal collateral is only slashed when a sector is terminated before the deal expires.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.client_collateral IS 'The amount of FIL (in attoFIL) the client has pledged as collateral.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.label IS 'An arbitrary client chosen label to apply to the deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.height IS 'Epoch at which this deal proposal was added or changed.';


-- ----------------------------------------------------------------
-- Name: market_deal_states
-- Model: market.MarketDealState
-- Growth: About 200 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.market_deal_states (
    deal_id bigint NOT NULL,
    sector_start_epoch bigint NOT NULL,
    last_update_epoch bigint NOT NULL,
    slash_epoch bigint NOT NULL,
    state_root text NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.market_deal_states ADD CONSTRAINT market_deal_states_pkey PRIMARY KEY (height, deal_id, state_root);
CREATE INDEX market_deal_states_height_idx ON {{ .SchemaName | default "public"}}.market_deal_states USING btree (height DESC);

-- Convert market_deal_states to a hypertable partitioned on height (time)
-- Assume ~200 per epoch, ~150 bytes per table row
-- Height chunked per 7 days so we expect 20160*200 = ~4032000 rows per chunk, ~576MiB per chunk
SELECT create_hypertable(
	'market_deal_states',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('market_deal_states', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.market_deal_states IS 'All storage deal state transitions detected on-chain.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_states.deal_id IS 'Identifier for the deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_states.sector_start_epoch IS 'Epoch this deal was included in a proven sector. -1 if not yet included in proven sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_states.last_update_epoch IS 'Epoch this deal was last updated at. -1 if deal state never updated.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_states.slash_epoch IS 'Epoch this deal was slashed at. -1 if deal was never slashed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_states.state_root IS 'CID of the parent state root for this deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_states.height IS 'Epoch at which this deal was added or changed.';

-- ----------------------------------------------------------------
-- Name: message_gas_economy
-- Model: messages.MessageGasEconomy
-- Growth: One row per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.message_gas_economy (
    state_root text NOT NULL,
    gas_limit_total numeric NOT NULL,
    gas_limit_unique_total numeric,
    base_fee numeric NOT NULL,
    base_fee_change_log double precision NOT NULL,
    gas_fill_ratio double precision,
    gas_capacity_ratio double precision,
    gas_waste_ratio double precision,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.message_gas_economy ADD CONSTRAINT message_gas_economy_pkey PRIMARY KEY (height, state_root);
CREATE INDEX message_gas_economy_height_idx ON {{ .SchemaName | default "public"}}.message_gas_economy USING btree (height DESC);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.message_gas_economy IS 'Gas economics for all messages in all blocks at each epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.gas_limit_total IS 'The sum of all the gas limits.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.gas_limit_unique_total IS 'The sum of all the gas limits of unique messages.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.base_fee IS 'The set price per unit of gas (measured in attoFIL/gas unit) to be burned (sent to an unrecoverable address) for every message execution.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.base_fee_change_log IS 'The logarithm of the change between new and old base fee.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.gas_fill_ratio IS 'The gas_limit_total / target gas limit total for all blocks.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.gas_capacity_ratio IS 'The gas_limit_unique_total / target gas limit total for all blocks.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.gas_waste_ratio IS '(gas_limit_total - gas_limit_unique_total) / target gas limit total for all blocks.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_gas_economy.height IS 'Epoch these economics apply to.';


-- ----------------------------------------------------------------
-- Name: messages
-- Model: messages.Message
-- Growth: About 400 rows per epoch
-- Notes: This was chunked daily in v0, now converted to weekly
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.messages (
    cid text NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    size_bytes bigint NOT NULL,
    nonce bigint NOT NULL,
    value numeric NOT NULL,
    gas_fee_cap numeric NOT NULL,
    gas_premium numeric NOT NULL,
    gas_limit bigint NOT NULL,
    method bigint,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.messages ADD CONSTRAINT messages_pkey PRIMARY KEY (height, cid);
CREATE INDEX messages_from_index ON {{ .SchemaName | default "public"}}.messages USING btree ("from");
CREATE INDEX messages_height_idx ON {{ .SchemaName | default "public"}}.messages USING btree (height DESC);
CREATE INDEX messages_to_index ON {{ .SchemaName | default "public"}}.messages USING btree ("to");

-- Convert messages to a hypertable partitioned on height (time)
-- Assume ~400 messages per epoch, ~373 bytes per table row (not including toast)
-- Height chunked per week so we expect 20160*400 = ~8064000 rows per chunk, ~2.8GiB per chunk
SELECT create_hypertable(
	'messages',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('messages', 'current_height', replace_if_exists => true);


COMMENT ON TABLE {{ .SchemaName | default "public"}}.messages IS 'Validated on-chain messages by their CID and their metadata.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages.cid IS 'CID of the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages."from" IS 'Address of the actor that sent the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages."to" IS 'Address of the actor that received the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages.size_bytes IS 'Size of the serialized message in bytes.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages.nonce IS 'The message nonce, which protects against duplicate messages and multiple messages with the same values.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages.value IS 'Amount of FIL (in attoFIL) transferred by this message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages.gas_fee_cap IS 'The maximum price that the message sender is willing to pay per unit of gas.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages.gas_premium IS 'The price per unit of gas (measured in attoFIL/gas) that the message sender is willing to pay (on top of the BaseFee) to "tip" the miner that will include this message in a block.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages.method IS 'The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.messages.height IS 'Epoch this message was executed at.';

-- ----------------------------------------------------------------
-- Name: miner_current_deadline_infos
-- Model: miner.MinerCurrentDeadlineInfo
-- Growth: About 1200 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_current_deadline_infos (
    height bigint NOT NULL,
    miner_id text NOT NULL,
    state_root text NOT NULL,
    deadline_index bigint NOT NULL,
    period_start bigint NOT NULL,
    open bigint NOT NULL,
    close bigint NOT NULL,
    challenge bigint NOT NULL,
    fault_cutoff bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_current_deadline_infos ADD CONSTRAINT miner_current_deadline_infos_pkey PRIMARY KEY (height, miner_id, state_root);
CREATE INDEX miner_current_deadline_infos_height_idx ON {{ .SchemaName | default "public"}}.miner_current_deadline_infos USING btree (height DESC);

SELECT create_hypertable(
	'miner_current_deadline_infos',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('miner_current_deadline_infos', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_current_deadline_infos IS 'Deadline refers to the window during which proofs may be submitted.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.height IS 'Epoch at which this info was calculated.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.miner_id IS 'Address of the miner this info relates to.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.deadline_index IS 'A deadline index, in [0..d.WPoStProvingPeriodDeadlines) unless period elapsed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.period_start IS 'First epoch of the proving period (<= CurrentEpoch).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.open IS 'First epoch from which a proof may be submitted (>= CurrentEpoch).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.close IS 'First epoch from which a proof may no longer be submitted (>= Open).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.challenge IS 'Epoch at which to sample the chain for challenge (< Open).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_current_deadline_infos.fault_cutoff IS 'First epoch at which a fault declaration is rejected (< Open).';


-- ----------------------------------------------------------------
-- Name: miner_fee_debts
-- Model: miner.MinerFeeDebt
-- Growth: About 1200 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_fee_debts (
    height bigint NOT NULL,
    miner_id text NOT NULL,
    state_root text NOT NULL,
    fee_debt numeric NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_fee_debts ADD CONSTRAINT miner_fee_debts_pkey PRIMARY KEY (height, miner_id, state_root);
CREATE INDEX miner_fee_debts_height_idx ON {{ .SchemaName | default "public"}}.miner_fee_debts USING btree (height DESC);

SELECT create_hypertable(
	'miner_fee_debts',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('miner_fee_debts', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_fee_debts IS 'Miner debts per epoch from unpaid fees.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_fee_debts.height IS 'Epoch at which this debt applies.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_fee_debts.miner_id IS 'Address of the miner that owes fees.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_fee_debts.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_fee_debts.fee_debt IS 'Absolute value of debt this miner owes from unpaid fees in attoFIL.';

-- ----------------------------------------------------------------
-- Name: miner_infos
-- Model: miner.MinerInfo
-- Growth: Less than one per epoch
-- Notes: This was a hypertable in v0, removed due to low rate of growth
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_infos (
    height bigint NOT NULL,
    miner_id text NOT NULL,
    state_root text NOT NULL,
    owner_id text NOT NULL,
    worker_id text NOT NULL,
    new_worker text,
    worker_change_epoch bigint NOT NULL,
    consensus_faulted_elapsed bigint NOT NULL,
    peer_id text,
    control_addresses jsonb,
    multi_addresses jsonb,
	sector_size bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_infos ADD CONSTRAINT miner_infos_pkey PRIMARY KEY (height, miner_id, state_root);
CREATE INDEX miner_infos_height_idx ON {{ .SchemaName | default "public"}}.miner_infos USING btree (height DESC);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_infos IS 'Miner Account IDs for all associated addresses plus peer ID. See https://docs.filecoin.io/mine/lotus/miner-addresses/ for more information.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.height IS 'Epoch at which this miner info was added/changed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.miner_id IS 'Address of miner this info applies to.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.owner_id IS 'Address of actor designated as the owner. The owner address is the address that created the miner, paid the collateral, and has block rewards paid out to it.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.worker_id IS 'Address of actor designated as the worker. The worker is responsible for doing all of the work, submitting proofs, committing new sectors, and all other day to day activities.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.new_worker IS 'Address of a new worker address that will become effective at worker_change_epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.worker_change_epoch IS 'Epoch at which a new_worker address will become effective.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.consensus_faulted_elapsed IS 'The next epoch this miner is eligible for certain permissioned actor methods and winning block elections as a result of being reported for a consensus fault.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.peer_id IS 'Current libp2p Peer ID of the miner.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.control_addresses IS 'JSON array of control addresses. Control addresses are used to submit WindowPoSts proofs to the chain. WindowPoSt is the mechanism through which storage is verified in Filecoin and is required by miners to submit proofs for all sectors every 24 hours. Those proofs are submitted as messages to the blockchain and therefore need to pay the respective fees.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_infos.multi_addresses IS 'JSON array of multiaddrs at which this miner can be reached.';


-- ----------------------------------------------------------------
-- Name: miner_locked_funds
-- Model: miner.MinerLockedFund
-- Growth: About 1200 per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_locked_funds (
    height bigint NOT NULL,
    miner_id text NOT NULL,
    state_root text NOT NULL,
    locked_funds numeric NOT NULL,
    initial_pledge numeric NOT NULL,
    pre_commit_deposits numeric NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_locked_funds ADD CONSTRAINT miner_locked_funds_pkey PRIMARY KEY (height, miner_id, state_root);
CREATE INDEX miner_locked_funds_height_idx ON {{ .SchemaName | default "public"}}.miner_locked_funds USING btree (height DESC);

SELECT create_hypertable(
	'miner_locked_funds',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('miner_locked_funds', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_locked_funds IS 'Details of Miner funds locked and unavailable for use.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_locked_funds.height IS 'Epoch at which these details were added/changed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_locked_funds.miner_id IS 'Address of the miner these details apply to.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_locked_funds.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_locked_funds.locked_funds IS 'Amount of FIL (in attoFIL) locked due to vesting. When a Miner receives tokens from block rewards, the tokens are locked and added to the Miner''s vesting table to be unlocked linearly over some future epochs.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_locked_funds.initial_pledge IS 'Amount of FIL (in attoFIL) locked due to it being pledged as collateral. When a Miner ProveCommits a Sector, they must supply an "initial pledge" for the Sector, which acts as collateral. If the Sector is terminated, this deposit is removed and burned along with rewards earned by this sector up to a limit.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_locked_funds.pre_commit_deposits IS 'Amount of FIL (in attoFIL) locked due to it being used as a PreCommit deposit. When a Miner PreCommits a Sector, they must supply a "precommit deposit" for the Sector, which acts as collateral. If the Sector is not ProveCommitted on time, this deposit is removed and burned.';


-- ----------------------------------------------------------------
-- Name: miner_pre_commit_infos
-- Model: MinerPreCommitInfo
-- Growth: About 180 per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_pre_commit_infos (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    state_root text NOT NULL,
    sealed_cid text NOT NULL,
    seal_rand_epoch bigint,
    expiration_epoch bigint,
    pre_commit_deposit numeric NOT NULL,
    pre_commit_epoch bigint,
    deal_weight numeric NOT NULL,
    verified_deal_weight numeric NOT NULL,
    is_replace_capacity boolean,
    replace_sector_deadline bigint,
    replace_sector_partition bigint,
    replace_sector_number bigint,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_pre_commit_infos ADD CONSTRAINT miner_pre_commit_infos_pkey PRIMARY KEY (height, miner_id, sector_id, state_root);
CREATE INDEX miner_pre_commit_infos_height_idx ON {{ .SchemaName | default "public"}}.miner_pre_commit_infos USING btree (height DESC);

-- Convert miner_pre_commit_infos to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~300 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, ~28MiB per chunk
SELECT create_hypertable(
	'miner_pre_commit_infos',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('miner_pre_commit_infos', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_pre_commit_infos IS 'Information on sector PreCommits.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.miner_id IS 'Address of the miner who owns the sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.sector_id IS 'Numeric identifier for the sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.sealed_cid IS 'CID of the sealed sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.seal_rand_epoch IS 'Seal challenge epoch. Epoch at which randomness should be drawn to tie Proof-of-Replication to a chain.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.expiration_epoch IS 'Epoch this sector expires.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.pre_commit_deposit IS 'Amount of FIL (in attoFIL) used as a PreCommit deposit. If the Sector is not ProveCommitted on time, this deposit is removed and burned.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.pre_commit_epoch IS 'Epoch this PreCommit was created.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.deal_weight IS 'Total space*time of submitted deals.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.verified_deal_weight IS 'Total space*time of submitted verified deals.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.is_replace_capacity IS 'Whether to replace a "committed capacity" no-deal sector (requires non-empty DealIDs).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.replace_sector_deadline IS 'The deadline location of the sector to replace.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.replace_sector_partition IS 'The partition location of the sector to replace.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.replace_sector_number IS 'ID of the committed capacity sector to replace.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_pre_commit_infos.height IS 'Epoch this PreCommit information was added/changed.';

-- ----------------------------------------------------------------
-- Name: miner_sector_deals
-- Model: MinerSectorDeal
-- Notes: This was a hypertable in v0, removed due to low rate of growth
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_sector_deals (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    deal_id bigint NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_sector_deals ADD CONSTRAINT miner_sector_deals_pkey PRIMARY KEY (height, miner_id, sector_id, deal_id);
CREATE INDEX miner_deal_sectors_height_idx ON {{ .SchemaName | default "public"}}.miner_sector_deals USING btree (height DESC);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_sector_deals IS 'Mapping of Deal IDs to their respective Miner and Sector IDs.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_deals.miner_id IS 'Address of the miner the deal is with.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_deals.sector_id IS 'Numeric identifier of the sector the deal is for.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_deals.deal_id IS 'Numeric identifier for the deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_deals.height IS 'Epoch at which this deal was added/updated.';


-- ----------------------------------------------------------------
-- Name: miner_sector_events
-- Model: miner.MinerSectorEvent
-- Growth: About 670 per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_sector_events (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    state_root text NOT NULL,
    event {{ .SchemaName | default "public"}}.miner_sector_event_type NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_sector_events ADD CONSTRAINT miner_sector_events_pkey PRIMARY KEY (height, sector_id, event, miner_id, state_root);
CREATE INDEX miner_sector_events_height_idx ON {{ .SchemaName | default "public"}}.miner_sector_events USING btree (height DESC);

-- Convert miner_sector_events to a hypertable partitioned on height (time)
-- Assume ~670 per epoch, ~300 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~13507200 rows per chunk, ~3.8GiB per chunk
SELECT create_hypertable(
	'miner_sector_events',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('miner_sector_events', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_sector_events IS 'Sector events on-chain per Miner/Sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_events.miner_id IS 'Address of the miner who owns the sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_events.sector_id IS 'Numeric identifier of the sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_events.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_events.event IS 'Name of the event that occurred.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_events.height IS 'Epoch at which this event occurred.';


-- ----------------------------------------------------------------
-- Name: miner_sector_infos
-- Model: miner.MinerSectorInfo
-- Growth: About 180 per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_sector_infos (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    state_root text NOT NULL,
    sealed_cid text NOT NULL,
    activation_epoch bigint,
    expiration_epoch bigint,
    deal_weight numeric NOT NULL,
    verified_deal_weight numeric NOT NULL,
    initial_pledge numeric NOT NULL,
    expected_day_reward numeric NOT NULL,
    expected_storage_pledge numeric NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_sector_infos ADD CONSTRAINT miner_sector_infos_pkey PRIMARY KEY (height, miner_id, sector_id, state_root);
CREATE INDEX miner_sector_infos_height_idx ON {{ .SchemaName | default "public"}}.miner_sector_infos USING btree (height DESC);

-- Convert miner_sector_infos to a hypertable partitioned on height (time)
-- Assume ~180 per epoch, ~300 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~3628800 rows per chunk, ~1GiB per chunk
SELECT create_hypertable(
	'miner_sector_infos',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('miner_sector_infos', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_sector_infos IS 'Latest state of sectors by Miner.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.miner_id IS 'Address of the miner who owns the sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.sector_id IS 'Numeric identifier of the sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.sealed_cid IS 'The root CID of the Sealed Sector’s merkle tree. Also called CommR, or "replica commitment".';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.activation_epoch IS 'Epoch during which the sector proof was accepted.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.expiration_epoch IS 'Epoch during which the sector expires.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.deal_weight IS 'Integral of active deals over sector lifetime.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.verified_deal_weight IS 'Integral of active verified deals over sector lifetime.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.initial_pledge IS 'Pledge collected to commit this sector (in attoFIL).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.expected_day_reward IS 'Expected one day projection of reward for sector computed at activation time (in attoFIL).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.expected_storage_pledge IS 'Expected twenty day projection of reward for sector computed at activation time (in attoFIL).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos.height IS 'Epoch at which this sector info was added/updated.';


-- ----------------------------------------------------------------
-- Name: miner_sector_posts
-- Model. miner.MinerSectorPost
-- Growth: About 9000 per epoch
-- Notes: This was chunked per 7 days in v0
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.miner_sector_posts (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    height bigint NOT NULL,
    post_message_cid text
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_sector_posts ADD CONSTRAINT miner_sector_posts_pkey PRIMARY KEY (height, miner_id, sector_id);
CREATE INDEX miner_sector_posts_height_idx ON {{ .SchemaName | default "public"}}.miner_sector_posts USING btree (height DESC);

-- Convert miner_sector_posts to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~150 bytes per table row
-- Height chunked per 7 days so we expect 2880*9000 = ~25920000 rows per chunk, ~3.7GiB per chunk
SELECT create_hypertable(
	'miner_sector_posts',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('miner_sector_posts', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_sector_posts IS 'Proof of Spacetime for sectors.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_posts.miner_id IS 'Address of the miner who owns the sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_posts.sector_id IS 'Numeric identifier of the sector.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_posts.height IS 'Epoch at which this PoSt message was executed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_posts.post_message_cid IS 'CID of the PoSt message.';


-- ----------------------------------------------------------------
-- Name: multisig_approvals
-- Model: msapprovals.MultisigApproval
-- Notes: This was a hypertable in v0, removed due to low rate of growth
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.multisig_approvals (
    height bigint NOT NULL,
    state_root text NOT NULL,
    multisig_id text NOT NULL,
    message text NOT NULL,
    method bigint NOT NULL,
    approver text NOT NULL,
    threshold bigint NOT NULL,
    initial_balance numeric NOT NULL,
    gas_used bigint NOT NULL,
    transaction_id bigint NOT NULL,
    "to" text NOT NULL,
    value numeric NOT NULL,
    signers jsonb NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.multisig_approvals ADD CONSTRAINT multisig_approvals_pkey PRIMARY KEY (height, state_root, multisig_id, message, approver);
CREATE INDEX multisig_approvals_height_idx ON {{ .SchemaName | default "public"}}.multisig_approvals USING btree (height DESC);

-- ----------------------------------------------------------------
-- Name: multisig_transactions
-- Model: MultisigTransaction
-- Growth: Less than 1 per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.multisig_transactions (
    height bigint NOT NULL,
    multisig_id text NOT NULL,
    state_root text NOT NULL,
    transaction_id bigint NOT NULL,
    "to" text NOT NULL,
    value text NOT NULL,
    method bigint NOT NULL,
    params bytea,
    approved jsonb NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.multisig_transactions ADD CONSTRAINT multisig_transactions_pkey PRIMARY KEY (height, state_root, multisig_id, transaction_id);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.multisig_transactions IS 'Details of pending transactions involving multisig actors.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions.height IS 'Epoch at which this transaction was executed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions.multisig_id IS 'Address of the multisig actor involved in the transaction.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions.transaction_id IS 'Number identifier for the transaction - unique per multisig.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions."to" IS 'Address of the recipient who will be sent a message if the proposal is approved.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions.value IS 'Amount of FIL (in attoFIL) that will be transferred if the proposal is approved.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions.method IS 'The method number to invoke on the recipient if the proposal is approved. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions.params IS 'CBOR encoded bytes of parameters to send to the method that will be invoked if the proposal is approved.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.multisig_transactions.approved IS 'Addresses of signers who have approved the transaction. 0th entry is the proposer.';


-- ----------------------------------------------------------------
-- Name: parsed_messages
-- Model: messages.ParsedMessage
-- Growth: About 400 per epoch
-- Notes: More accurate chunk size calculation based on actual row sizes
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.parsed_messages (
    cid text NOT NULL,
    height bigint NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    value numeric NOT NULL,
    method text NOT NULL,
    params jsonb
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.parsed_messages ADD CONSTRAINT parsed_messages_pkey PRIMARY KEY (height, cid);
CREATE INDEX parsed_messages_height_idx ON {{ .SchemaName | default "public"}}.parsed_messages USING btree (height DESC);
CREATE INDEX message_parsed_from_idx ON {{ .SchemaName | default "public"}}.parsed_messages USING hash ("from");
CREATE INDEX message_parsed_method_idx ON {{ .SchemaName | default "public"}}.parsed_messages USING hash (method);
CREATE INDEX message_parsed_to_idx ON {{ .SchemaName | default "public"}}.parsed_messages USING hash ("to");

-- Convert messages to a hypertable partitioned on height (time)
-- Assume ~400 messages per epoch, ~2500 bytes per table row
-- Height chunked per day so we expect 2880*400 = ~1152000 rows per chunk, ~2.7GiB per chunk
SELECT create_hypertable(
	'parsed_messages',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('parsed_messages', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.parsed_messages IS 'Messages parsed to extract useful information.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.parsed_messages.cid IS 'CID of the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.parsed_messages.height IS 'Epoch this message was executed at.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.parsed_messages."from" IS 'Address of the actor that sent the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.parsed_messages."to" IS 'Address of the actor that received the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.parsed_messages.value IS 'Amount of FIL (in attoFIL) transferred by this message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.parsed_messages.method IS 'The name of the method that was invoked on the recipient actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.parsed_messages.params IS 'Method parameters parsed and serialized as a JSON object.';


-- ----------------------------------------------------------------
-- Name: power_actor_claims
-- Model: power.PowerActorClaim
-- Growth: About 7 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.power_actor_claims (
    height bigint NOT NULL,
    miner_id text NOT NULL,
    state_root text NOT NULL,
    raw_byte_power numeric NOT NULL,
    quality_adj_power numeric NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.power_actor_claims ADD CONSTRAINT power_actor_claims_pkey PRIMARY KEY (height, miner_id, state_root);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.power_actor_claims IS 'Miner power claims recorded by the power actor.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.power_actor_claims.height IS 'Epoch this claim was made.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.power_actor_claims.miner_id IS 'Address of miner making the claim.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.power_actor_claims.state_root IS 'CID of the parent state root at this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.power_actor_claims.raw_byte_power IS 'Sum of raw byte storage power for a miner''s sectors. Raw byte power is the size of a sector in bytes.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.power_actor_claims.quality_adj_power IS 'Sum of quality adjusted storage power for a miner''s sectors. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals.';


-- ----------------------------------------------------------------
-- Name: receipts
-- Model: messages.Receipt
-- Growth: About 400 per epoch
-- Notes: This was chunked daily in v0, now converted to weekly
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.receipts (
    message text NOT NULL,
    state_root text NOT NULL,
    idx bigint NOT NULL,
    exit_code bigint NOT NULL,
    gas_used bigint NOT NULL,
    height bigint NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.receipts ADD CONSTRAINT receipts_pkey PRIMARY KEY (height, message, state_root);
CREATE INDEX receipts_height_idx ON {{ .SchemaName | default "public"}}.receipts USING btree (height DESC);

-- Convert receipts to a hypertable partitioned on height (time)
-- Assume ~400 receipts per epoch, ~215 bytes per table row
-- Height chunked per day so we expect 20160*250 = ~8064000 rows per chunk, ~1.6GiB per chunk
SELECT create_hypertable(
	'receipts',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('receipts', 'current_height', replace_if_exists => true);

COMMENT ON TABLE {{ .SchemaName | default "public"}}.receipts IS 'Message reciepts after being applied to chain state by message CID and parent state root CID of tipset when message was executed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipts.message IS 'CID of the message this receipt belongs to.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipts.state_root IS 'CID of the parent state root that this epoch.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipts.idx IS 'Index of message indicating execution order.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipts.exit_code IS 'The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipts.gas_used IS 'A measure of the amount of resources (or units of gas) consumed, in order to execute a message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipts.height IS 'Epoch the message was executed and receipt generated.';


-- ----------------------------------------------------------------
-- Name: visor_processing_reports
-- Model: visor.ProcessingReport
-- Growth: About 8 per epoch
-- ----------------------------------------------------------------

CREATE TABLE {{ .SchemaName | default "public"}}.visor_processing_reports (
    height bigint NOT NULL,
    state_root text NOT NULL,
    reporter text NOT NULL,
    task text NOT NULL,
    started_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone NOT NULL,
    status text,
    status_information text,
    errors_detected jsonb
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.visor_processing_reports ADD CONSTRAINT visor_processing_reports_pkey PRIMARY KEY (height, state_root, reporter, task, started_at);


-- ----------------------------------------------------------------
-- Name: visor_version
-- Notes: This table can be created during version checking before a migration.
-- ----------------------------------------------------------------

CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.visor_version (
    major integer NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.visor_version DROP CONSTRAINT IF EXISTS visor_version_pkey;
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.visor_version ADD CONSTRAINT visor_version_pkey PRIMARY KEY (major);
INSERT INTO {{ .SchemaName | default "public"}}.visor_version (major) VALUES (1);


-- =====================================================================================================================
-- VIEWS
-- =====================================================================================================================

--
-- Name: chain_visualizer_blocks_view
--

CREATE VIEW {{ .SchemaName | default "public"}}.chain_visualizer_blocks_view AS
 SELECT block_headers.cid,
    block_headers.parent_weight,
    block_headers.parent_state_root,
    block_headers.height,
    block_headers.miner,
    block_headers."timestamp",
    block_headers.win_count,
    block_headers.parent_base_fee,
    block_headers.fork_signaling
   FROM {{ .SchemaName | default "public"}}.block_headers;


--
-- Name: chain_visualizer_blocks_with_parents_view
--

CREATE VIEW {{ .SchemaName | default "public"}}.chain_visualizer_blocks_with_parents_view AS
 SELECT block_parents.block,
    block_parents.parent,
    b.miner,
    b.height,
    b."timestamp"
   FROM ({{ .SchemaName | default "public"}}.block_parents
     JOIN {{ .SchemaName | default "public"}}.block_headers b ON ((block_parents.block = b.cid)));

--
-- Name: chain_visualizer_chain_data_view; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW {{ .SchemaName | default "public"}}.chain_visualizer_chain_data_view AS
 SELECT main_block.cid AS block,
    bp.parent,
    main_block.miner,
    main_block.height,
    main_block.parent_weight AS parentweight,
    main_block."timestamp",
    main_block.parent_state_root AS parentstateroot,
    parent_block."timestamp" AS parenttimestamp,
    parent_block.height AS parentheight,
    pac.raw_byte_power AS parentpower,
    main_block."timestamp" AS syncedtimestamp,
    ( SELECT count(*) AS count
           FROM {{ .SchemaName | default "public"}}.block_messages
          WHERE (block_messages.block = main_block.cid)) AS messages
   FROM ((({{ .SchemaName | default "public"}}.block_headers main_block
     LEFT JOIN {{ .SchemaName | default "public"}}.block_parents bp ON ((bp.block = main_block.cid)))
     LEFT JOIN {{ .SchemaName | default "public"}}.block_headers parent_block ON ((parent_block.cid = bp.parent)))
     LEFT JOIN {{ .SchemaName | default "public"}}.power_actor_claims pac ON ((main_block.parent_state_root = pac.state_root)));

--
-- Name: chain_visualizer_orphans_view; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW {{ .SchemaName | default "public"}}.chain_visualizer_orphans_view AS
 SELECT block_headers.cid AS block,
    block_headers.miner,
    block_headers.height,
    block_headers.parent_weight AS parentweight,
    block_headers."timestamp",
    block_headers.parent_state_root AS parentstateroot,
    block_parents.parent
   FROM ({{ .SchemaName | default "public"}}.block_headers
     LEFT JOIN {{ .SchemaName | default "public"}}.block_parents ON ((block_headers.cid = block_parents.parent)))
  WHERE (block_parents.block IS NULL);

--
-- Name: derived_consensus_chain_view; Type: MATERIALIZED VIEW; Schema: public; Owner: postgres
--

CREATE MATERIALIZED VIEW {{ .SchemaName | default "public"}}.derived_consensus_chain_view AS
 WITH RECURSIVE consensus_chain AS (
         SELECT b.cid,
            b.height,
            b.miner,
            b."timestamp",
            b.parent_state_root,
            b.win_count
           FROM {{ .SchemaName | default "public"}}.block_headers b
          WHERE (b.parent_state_root = ( SELECT block_headers.parent_state_root
                   FROM {{ .SchemaName | default "public"}}.block_headers
                  ORDER BY block_headers.height DESC, block_headers.parent_weight DESC
                 LIMIT 1))
        UNION
         SELECT p.cid,
            p.height,
            p.miner,
            p."timestamp",
            p.parent_state_root,
            p.win_count
           FROM (({{ .SchemaName | default "public"}}.block_headers p
             JOIN {{ .SchemaName | default "public"}}.block_parents pb ON ((p.cid = pb.parent)))
             JOIN consensus_chain c ON ((c.cid = pb.block)))
        )
 SELECT consensus_chain.cid,
    consensus_chain.height,
    consensus_chain.miner,
    consensus_chain."timestamp",
    consensus_chain.parent_state_root,
    consensus_chain.win_count
   FROM consensus_chain
  WITH NO DATA;


--
-- Name: state_heights; Type: MATERIALIZED VIEW; Schema: public; Owner: postgres
--

CREATE MATERIALIZED VIEW {{ .SchemaName | default "public"}}.state_heights AS
 SELECT DISTINCT block_headers.height,
    block_headers.parent_state_root AS parentstateroot
   FROM {{ .SchemaName | default "public"}}.block_headers
  WITH NO DATA;
CREATE INDEX state_heights_height_index ON {{ .SchemaName | default "public"}}.state_heights USING btree (height);
CREATE INDEX state_heights_parentstateroot_index ON {{ .SchemaName | default "public"}}.state_heights USING btree (parentstateroot);

`
