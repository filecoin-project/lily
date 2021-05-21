package v1

import (
	"github.com/filecoin-project/sentinel-visor/schemas"
	"github.com/go-pg/migrations/v8"
)

// Patches is the collection of patches made to the base schema
var Patches = migrations.NewCollection()

func init() {
	schemas.RegisterSchema(1)
}

// Base is the initial schema for this major version. Patches are applied on top of this base.
var Base = `
--
-- PostgreSQL database dump
--

--
-- Name: timescaledb; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS timescaledb WITH SCHEMA public;


--
-- Name: EXTENSION timescaledb; Type: COMMENT; Schema: -; Owner:
--

COMMENT ON EXTENSION timescaledb IS 'Enables scalable inserts and complex queries for time-series data';


--
-- Name: miner_sector_event_type; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public.miner_sector_event_type AS ENUM (
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


ALTER TYPE public.miner_sector_event_type OWNER TO postgres;

--
-- Name: actor_tips(bigint); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.actor_tips(epoch bigint) RETURNS TABLE(id text, code text, head text, nonce integer, balance text, stateroot text, height bigint, parentstateroot text)
    LANGUAGE sql
    AS $_$
    select distinct on (id) * from actors
        inner join state_heights sh on sh.parentstateroot = stateroot
        where height < $1
		order by id, height desc;
$_$;


ALTER FUNCTION public.actor_tips(epoch bigint) OWNER TO postgres;

--
-- Name: height_to_unix(bigint); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.height_to_unix(fil_epoch bigint) RETURNS bigint
    LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
    AS $$
		SELECT ((fil_epoch * 30) + 1598306400)::bigint;
	$$;


ALTER FUNCTION public.height_to_unix(fil_epoch bigint) OWNER TO postgres;

--
-- Name: unix_to_height(bigint); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.unix_to_height(unix_epoch bigint) RETURNS bigint
    LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
    AS $$
		SELECT ((unix_epoch - 1598306400) / 30)::bigint;
	$$;


ALTER FUNCTION public.unix_to_height(unix_epoch bigint) OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: actor_states; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.actor_states (
    head text NOT NULL,
    code text NOT NULL,
    state jsonb NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.actor_states OWNER TO postgres;

--
-- Name: TABLE actor_states; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.actor_states IS 'Actor states that were changed at an epoch. Associates actors states as single-level trees with CIDs pointing to complete state tree with the root CID (head) for that actor''s state.';


--
-- Name: COLUMN actor_states.head; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actor_states.head IS 'CID of the root of the state tree for the actor.';


--
-- Name: COLUMN actor_states.code; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actor_states.code IS 'CID identifier for the type of the actor.';


--
-- Name: COLUMN actor_states.state; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actor_states.state IS 'Top level of state data.';


--
-- Name: COLUMN actor_states.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actor_states.height IS 'Epoch when this state change happened.';


--
-- Name: actors; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.actors (
    id text NOT NULL,
    code text NOT NULL,
    head text NOT NULL,
    nonce bigint NOT NULL,
    balance text NOT NULL,
    state_root text NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.actors OWNER TO postgres;

--
-- Name: TABLE actors; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.actors IS 'Actors on chain that were added or updated at an epoch. Associates the actor''s state root CID (head) with the chain state root CID from which it decends. Includes account ID nonce and balance at each state.';


--
-- Name: COLUMN actors.id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actors.id IS 'Actor address.';


--
-- Name: COLUMN actors.code; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actors.code IS 'Human readable identifier for the type of the actor.';


--
-- Name: COLUMN actors.head; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actors.head IS 'CID of the root of the state tree for the actor.';


--
-- Name: COLUMN actors.nonce; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actors.nonce IS 'The next actor nonce that is expected to appear on chain.';


--
-- Name: COLUMN actors.balance; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actors.balance IS 'Actor balance in attoFIL.';


--
-- Name: COLUMN actors.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actors.state_root IS 'CID of the state root.';


--
-- Name: COLUMN actors.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.actors.height IS 'Epoch when this actor was created or updated.';


--
-- Name: block_headers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.block_headers (
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


ALTER TABLE public.block_headers OWNER TO postgres;

--
-- Name: TABLE block_headers; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.block_headers IS 'Blocks included in tipsets at an epoch.';


--
-- Name: COLUMN block_headers.cid; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers.cid IS 'CID of the block.';


--
-- Name: COLUMN block_headers.parent_weight; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers.parent_weight IS 'Aggregate chain weight of the block''s parent set.';


--
-- Name: COLUMN block_headers.parent_state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers.parent_state_root IS 'CID of the block''s parent state root.';


--
-- Name: COLUMN block_headers.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers.height IS 'Epoch when this block was mined.';


--
-- Name: COLUMN block_headers.miner; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers.miner IS 'Address of the miner who mined this block.';


--
-- Name: COLUMN block_headers."timestamp"; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers."timestamp" IS 'Time the block was mined in Unix time, the number of seconds elapsed since January 1, 1970 UTC.';


--
-- Name: COLUMN block_headers.win_count; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers.win_count IS 'Number of reward units won in this block.';


--
-- Name: COLUMN block_headers.parent_base_fee; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers.parent_base_fee IS 'The base fee after executing the parent tipset.';


--
-- Name: COLUMN block_headers.fork_signaling; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_headers.fork_signaling IS 'Flag used as part of signaling forks.';


--
-- Name: block_messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.block_messages (
    block text NOT NULL,
    message text NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.block_messages OWNER TO postgres;

--
-- Name: TABLE block_messages; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.block_messages IS 'Message CIDs and the Blocks CID which contain them.';


--
-- Name: COLUMN block_messages.block; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_messages.block IS 'CID of the block that contains the message.';


--
-- Name: COLUMN block_messages.message; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_messages.message IS 'CID of a message in the block.';


--
-- Name: COLUMN block_messages.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_messages.height IS 'Epoch when the block was mined.';


--
-- Name: block_parents; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.block_parents (
    block text NOT NULL,
    parent text NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.block_parents OWNER TO postgres;

--
-- Name: TABLE block_parents; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.block_parents IS 'Block CIDs to many parent Block CIDs.';


--
-- Name: COLUMN block_parents.block; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_parents.block IS 'CID of the block.';


--
-- Name: COLUMN block_parents.parent; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_parents.parent IS 'CID of the parent block.';


--
-- Name: COLUMN block_parents.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.block_parents.height IS 'Epoch when the block was mined.';


--
-- Name: blocks_synced; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.blocks_synced (
    cid text NOT NULL,
    synced_at timestamp with time zone NOT NULL,
    processed_at timestamp with time zone,
    height bigint,
    completed_at timestamp with time zone
);


ALTER TABLE public.blocks_synced OWNER TO postgres;

--
-- Name: chain_economics; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chain_economics (
    parent_state_root text NOT NULL,
    circulating_fil text NOT NULL,
    vested_fil text NOT NULL,
    mined_fil text NOT NULL,
    burnt_fil text NOT NULL,
    locked_fil text NOT NULL
);


ALTER TABLE public.chain_economics OWNER TO postgres;

--
-- Name: TABLE chain_economics; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.chain_economics IS 'Economic summaries per state root CID.';


--
-- Name: COLUMN chain_economics.parent_state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_economics.parent_state_root IS 'CID of the parent state root.';


--
-- Name: COLUMN chain_economics.circulating_fil; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_economics.circulating_fil IS 'The amount of FIL (attoFIL) circulating and tradeable in the economy. The basis for Market Cap calculations.';


--
-- Name: COLUMN chain_economics.vested_fil; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_economics.vested_fil IS 'Total amount of FIL (attoFIL) that is vested from genesis allocation.';


--
-- Name: COLUMN chain_economics.mined_fil; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_economics.mined_fil IS 'The amount of FIL (attoFIL) that has been mined by storage miners.';


--
-- Name: COLUMN chain_economics.burnt_fil; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_economics.burnt_fil IS 'Total FIL (attoFIL) burned as part of penalties and on-chain computations.';


--
-- Name: COLUMN chain_economics.locked_fil; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_economics.locked_fil IS 'The amount of FIL (attoFIL) locked as part of mining, deals, and other mechanisms.';


--
-- Name: chain_powers; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chain_powers (
    state_root text NOT NULL,
    total_raw_bytes_power text NOT NULL,
    total_raw_bytes_committed text NOT NULL,
    total_qa_bytes_power text NOT NULL,
    total_qa_bytes_committed text NOT NULL,
    total_pledge_collateral text NOT NULL,
    qa_smoothed_position_estimate text NOT NULL,
    qa_smoothed_velocity_estimate text NOT NULL,
    miner_count bigint,
    participating_miner_count bigint,
    height bigint NOT NULL
);


ALTER TABLE public.chain_powers OWNER TO postgres;

--
-- Name: TABLE chain_powers; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.chain_powers IS 'Power summaries from the Power actor.';


--
-- Name: COLUMN chain_powers.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.state_root IS 'CID of the parent state root.';


--
-- Name: COLUMN chain_powers.total_raw_bytes_power; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.total_raw_bytes_power IS 'Total storage power in bytes in the network. Raw byte power is the size of a sector in bytes.';


--
-- Name: COLUMN chain_powers.total_raw_bytes_committed; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.total_raw_bytes_committed IS 'Total provably committed storage power in bytes. Raw byte power is the size of a sector in bytes.';


--
-- Name: COLUMN chain_powers.total_qa_bytes_power; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.total_qa_bytes_power IS 'Total quality adjusted storage power in bytes in the network. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals.';


--
-- Name: COLUMN chain_powers.total_qa_bytes_committed; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.total_qa_bytes_committed IS 'Total provably committed, quality adjusted storage power in bytes. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals.';


--
-- Name: COLUMN chain_powers.total_pledge_collateral; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.total_pledge_collateral IS 'Total locked FIL (attoFIL) miners have pledged as collateral in order to participate in the economy.';


--
-- Name: COLUMN chain_powers.qa_smoothed_position_estimate; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.qa_smoothed_position_estimate IS 'Total power smoothed position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format.';


--
-- Name: COLUMN chain_powers.qa_smoothed_velocity_estimate; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.qa_smoothed_velocity_estimate IS 'Total power smoothed velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format.';


--
-- Name: COLUMN chain_powers.miner_count; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.miner_count IS 'Total number of miners.';


--
-- Name: COLUMN chain_powers.participating_miner_count; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.participating_miner_count IS 'Total number of miners with power above the minimum miner threshold.';


--
-- Name: COLUMN chain_powers.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_powers.height IS 'Epoch this power summary applies to.';


--
-- Name: chain_rewards; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chain_rewards (
    state_root text NOT NULL,
    cum_sum_baseline text NOT NULL,
    cum_sum_realized text NOT NULL,
    effective_baseline_power text NOT NULL,
    new_baseline_power text NOT NULL,
    new_reward_smoothed_position_estimate text NOT NULL,
    new_reward_smoothed_velocity_estimate text NOT NULL,
    total_mined_reward text NOT NULL,
    new_reward text,
    effective_network_time bigint,
    height bigint NOT NULL
);


ALTER TABLE public.chain_rewards OWNER TO postgres;

--
-- Name: TABLE chain_rewards; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.chain_rewards IS 'Reward summaries from the Reward actor.';


--
-- Name: COLUMN chain_rewards.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.state_root IS 'CID of the parent state root.';


--
-- Name: COLUMN chain_rewards.cum_sum_baseline; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.cum_sum_baseline IS 'Target that CumsumRealized needs to reach for EffectiveNetworkTime to increase. It is measured in byte-epochs (space * time) representing power committed to the network for some duration.';


--
-- Name: COLUMN chain_rewards.cum_sum_realized; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.cum_sum_realized IS 'Cumulative sum of network power capped by BaselinePower(epoch). It is measured in byte-epochs (space * time) representing power committed to the network for some duration.';


--
-- Name: COLUMN chain_rewards.effective_baseline_power; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.effective_baseline_power IS 'The baseline power (in bytes) at the EffectiveNetworkTime epoch.';


--
-- Name: COLUMN chain_rewards.new_baseline_power; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.new_baseline_power IS 'The baseline power (in bytes) the network is targeting.';


--
-- Name: COLUMN chain_rewards.new_reward_smoothed_position_estimate; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.new_reward_smoothed_position_estimate IS 'Smoothed reward position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format.';


--
-- Name: COLUMN chain_rewards.new_reward_smoothed_velocity_estimate; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.new_reward_smoothed_velocity_estimate IS 'Smoothed reward velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format.';


--
-- Name: COLUMN chain_rewards.total_mined_reward; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.total_mined_reward IS 'The total FIL (attoFIL) awarded to block miners.';


--
-- Name: COLUMN chain_rewards.new_reward; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.new_reward IS 'The reward to be paid in per WinCount to block producers. The actual reward total paid out depends on the number of winners in any round. This value is recomputed every non-null epoch and used in the next non-null epoch.';


--
-- Name: COLUMN chain_rewards.effective_network_time; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.effective_network_time IS 'Ceiling of real effective network time "theta" based on CumsumBaselinePower(theta) == CumsumRealizedPower. Theta captures the notion of how much the network has progressed in its baseline and in advancing network time.';


--
-- Name: COLUMN chain_rewards.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.chain_rewards.height IS 'Epoch this rewards summary applies to.';


--
-- Name: chain_visualizer_blocks_view; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.chain_visualizer_blocks_view AS
 SELECT block_headers.cid,
    block_headers.parent_weight,
    block_headers.parent_state_root,
    block_headers.height,
    block_headers.miner,
    block_headers."timestamp",
    block_headers.win_count,
    block_headers.parent_base_fee,
    block_headers.fork_signaling
   FROM public.block_headers;


ALTER TABLE public.chain_visualizer_blocks_view OWNER TO postgres;

--
-- Name: chain_visualizer_blocks_with_parents_view; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.chain_visualizer_blocks_with_parents_view AS
 SELECT block_parents.block,
    block_parents.parent,
    b.miner,
    b.height,
    b."timestamp"
   FROM (public.block_parents
     JOIN public.block_headers b ON ((block_parents.block = b.cid)));


ALTER TABLE public.chain_visualizer_blocks_with_parents_view OWNER TO postgres;

--
-- Name: power_actor_claims; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.power_actor_claims (
    height bigint NOT NULL,
    miner_id text NOT NULL,
    state_root text NOT NULL,
    raw_byte_power text NOT NULL,
    quality_adj_power text NOT NULL
);


ALTER TABLE public.power_actor_claims OWNER TO postgres;

--
-- Name: TABLE power_actor_claims; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.power_actor_claims IS 'Miner power claims recorded by the power actor.';


--
-- Name: COLUMN power_actor_claims.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.power_actor_claims.height IS 'Epoch this claim was made.';


--
-- Name: COLUMN power_actor_claims.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.power_actor_claims.miner_id IS 'Address of miner making the claim.';


--
-- Name: COLUMN power_actor_claims.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.power_actor_claims.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN power_actor_claims.raw_byte_power; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.power_actor_claims.raw_byte_power IS 'Sum of raw byte storage power for a miner''s sectors. Raw byte power is the size of a sector in bytes.';


--
-- Name: COLUMN power_actor_claims.quality_adj_power; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.power_actor_claims.quality_adj_power IS 'Sum of quality adjusted storage power for a miner''s sectors. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals.';


--
-- Name: chain_visualizer_chain_data_view; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.chain_visualizer_chain_data_view AS
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
           FROM public.block_messages
          WHERE (block_messages.block = main_block.cid)) AS messages
   FROM (((public.block_headers main_block
     LEFT JOIN public.block_parents bp ON ((bp.block = main_block.cid)))
     LEFT JOIN public.block_headers parent_block ON ((parent_block.cid = bp.parent)))
     LEFT JOIN public.power_actor_claims pac ON ((main_block.parent_state_root = pac.state_root)));


ALTER TABLE public.chain_visualizer_chain_data_view OWNER TO postgres;

--
-- Name: chain_visualizer_orphans_view; Type: VIEW; Schema: public; Owner: postgres
--

CREATE VIEW public.chain_visualizer_orphans_view AS
 SELECT block_headers.cid AS block,
    block_headers.miner,
    block_headers.height,
    block_headers.parent_weight AS parentweight,
    block_headers."timestamp",
    block_headers.parent_state_root AS parentstateroot,
    block_parents.parent
   FROM (public.block_headers
     LEFT JOIN public.block_parents ON ((block_headers.cid = block_parents.parent)))
  WHERE (block_parents.block IS NULL);


ALTER TABLE public.chain_visualizer_orphans_view OWNER TO postgres;

--
-- Name: derived_consensus_chain_view; Type: MATERIALIZED VIEW; Schema: public; Owner: postgres
--

CREATE MATERIALIZED VIEW public.derived_consensus_chain_view AS
 WITH RECURSIVE consensus_chain AS (
         SELECT b.cid,
            b.height,
            b.miner,
            b."timestamp",
            b.parent_state_root,
            b.win_count
           FROM public.block_headers b
          WHERE (b.parent_state_root = ( SELECT block_headers.parent_state_root
                   FROM public.block_headers
                  ORDER BY block_headers.height DESC, block_headers.parent_weight DESC
                 LIMIT 1))
        UNION
         SELECT p.cid,
            p.height,
            p.miner,
            p."timestamp",
            p.parent_state_root,
            p.win_count
           FROM ((public.block_headers p
             JOIN public.block_parents pb ON ((p.cid = pb.parent)))
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


ALTER TABLE public.derived_consensus_chain_view OWNER TO postgres;

--
-- Name: derived_gas_outputs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.derived_gas_outputs (
    cid text NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    value text NOT NULL,
    gas_fee_cap text NOT NULL,
    gas_premium text NOT NULL,
    gas_limit bigint,
    size_bytes bigint,
    nonce bigint,
    method bigint,
    state_root text NOT NULL,
    exit_code bigint NOT NULL,
    gas_used bigint NOT NULL,
    parent_base_fee text NOT NULL,
    base_fee_burn text NOT NULL,
    over_estimation_burn text NOT NULL,
    miner_penalty text NOT NULL,
    miner_tip text NOT NULL,
    refund text NOT NULL,
    gas_refund bigint NOT NULL,
    gas_burned bigint NOT NULL,
    height bigint NOT NULL,
    actor_name text NOT NULL
);


ALTER TABLE public.derived_gas_outputs OWNER TO postgres;

--
-- Name: TABLE derived_gas_outputs; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.derived_gas_outputs IS 'Derived gas costs resulting from execution of a message in the VM.';


--
-- Name: COLUMN derived_gas_outputs.cid; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.cid IS 'CID of the message.';


--
-- Name: COLUMN derived_gas_outputs."from"; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs."from" IS 'Address of actor that sent the message.';


--
-- Name: COLUMN derived_gas_outputs."to"; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs."to" IS 'Address of actor that received the message.';


--
-- Name: COLUMN derived_gas_outputs.value; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.value IS 'The FIL value transferred (attoFIL) to the message receiver.';


--
-- Name: COLUMN derived_gas_outputs.gas_fee_cap; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.gas_fee_cap IS 'The maximum price that the message sender is willing to pay per unit of gas.';


--
-- Name: COLUMN derived_gas_outputs.gas_premium; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.gas_premium IS 'The price per unit of gas (measured in attoFIL/gas) that the message sender is willing to pay (on top of the BaseFee) to "tip" the miner that will include this message in a block.';


--
-- Name: COLUMN derived_gas_outputs.gas_limit; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.gas_limit IS 'A hard limit on the amount of gas (i.e., number of units of gas) that a message’s execution should be allowed to consume on chain. It is measured in units of gas.';


--
-- Name: COLUMN derived_gas_outputs.size_bytes; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.size_bytes IS 'Size in bytes of the serialized message.';


--
-- Name: COLUMN derived_gas_outputs.nonce; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.nonce IS 'The message nonce, which protects against duplicate messages and multiple messages with the same values.';


--
-- Name: COLUMN derived_gas_outputs.method; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.method IS 'The method number to invoke. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';


--
-- Name: COLUMN derived_gas_outputs.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.state_root IS 'CID of the parent state root.';


--
-- Name: COLUMN derived_gas_outputs.exit_code; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.exit_code IS 'The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific.';


--
-- Name: COLUMN derived_gas_outputs.gas_used; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.gas_used IS 'A measure of the amount of resources (or units of gas) consumed, in order to execute a message.';


--
-- Name: COLUMN derived_gas_outputs.parent_base_fee; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.parent_base_fee IS 'The set price per unit of gas (measured in attoFIL/gas unit) to be burned (sent to an unrecoverable address) for every message execution.';


--
-- Name: COLUMN derived_gas_outputs.base_fee_burn; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.base_fee_burn IS 'The amount of FIL (in attoFIL) to burn as a result of the base fee. It is parent_base_fee (or gas_fee_cap if smaller) multiplied by gas_used. Note: successfull window PoSt messages are not charged this burn.';


--
-- Name: COLUMN derived_gas_outputs.over_estimation_burn; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.over_estimation_burn IS 'The fee to pay (in attoFIL) for overestimating the gas used to execute a message. The overestimated gas to burn (gas_burned) is a portion of the difference between gas_limit and gas_used. The over_estimation_burn value is gas_burned * parent_base_fee.';


--
-- Name: COLUMN derived_gas_outputs.miner_penalty; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.miner_penalty IS 'Any penalty fees (in attoFIL) the miner incured while executing the message.';


--
-- Name: COLUMN derived_gas_outputs.miner_tip; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.miner_tip IS 'The amount of FIL (in attoFIL) the miner receives for executing the message. Typically it is gas_premium * gas_limit but may be lower if the total fees exceed the gas_fee_cap.';


--
-- Name: COLUMN derived_gas_outputs.refund; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.refund IS 'The amount of FIL (in attoFIL) to refund to the message sender after base fee, miner tip and overestimation amounts have been deducted.';


--
-- Name: COLUMN derived_gas_outputs.gas_refund; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.gas_refund IS 'The overestimated units of gas to refund. It is a portion of the difference between gas_limit and gas_used.';


--
-- Name: COLUMN derived_gas_outputs.gas_burned; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.gas_burned IS 'The overestimated units of gas to burn. It is a portion of the difference between gas_limit and gas_used.';


--
-- Name: COLUMN derived_gas_outputs.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.height IS 'Epoch this message was executed at.';


--
-- Name: COLUMN derived_gas_outputs.actor_name; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.derived_gas_outputs.actor_name IS 'Human readable identifier for the type of the actor.';


--
-- Name: drand_block_entries; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.drand_block_entries (
    round bigint NOT NULL,
    block text NOT NULL
);


ALTER TABLE public.drand_block_entries OWNER TO postgres;

--
-- Name: TABLE drand_block_entries; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.drand_block_entries IS 'Drand randomness round numbers used in each block.';


--
-- Name: COLUMN drand_block_entries.round; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.drand_block_entries.round IS 'The round number of the randomness used.';


--
-- Name: COLUMN drand_block_entries.block; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.drand_block_entries.block IS 'CID of the block.';


--
-- Name: gopg_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.gopg_migrations (
    id integer NOT NULL,
    version bigint,
    created_at timestamp with time zone
);


ALTER TABLE public.gopg_migrations OWNER TO postgres;

--
-- Name: gopg_migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.gopg_migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.gopg_migrations_id_seq OWNER TO postgres;

--
-- Name: gopg_migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.gopg_migrations_id_seq OWNED BY public.gopg_migrations.id;


--
-- Name: id_address_map; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.id_address_map (
    id text NOT NULL,
    address text NOT NULL
);


ALTER TABLE public.id_address_map OWNER TO postgres;

--
-- Name: id_addresses; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.id_addresses (
    id text NOT NULL,
    address text NOT NULL,
    state_root text NOT NULL
);


ALTER TABLE public.id_addresses OWNER TO postgres;

--
-- Name: TABLE id_addresses; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.id_addresses IS 'Mapping of IDs to robust addresses from the init actor''s state.';


--
-- Name: COLUMN id_addresses.id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.id_addresses.id IS 'ID of the actor.';


--
-- Name: COLUMN id_addresses.address; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.id_addresses.address IS 'Robust address of the actor.';


--
-- Name: COLUMN id_addresses.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.id_addresses.state_root IS 'CID of the parent state root at which this address mapping was added.';


--
-- Name: market_deal_proposals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.market_deal_proposals (
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


ALTER TABLE public.market_deal_proposals OWNER TO postgres;

--
-- Name: TABLE market_deal_proposals; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.market_deal_proposals IS 'All storage deal states with latest values applied to end_epoch when updates are detected on-chain.';


--
-- Name: COLUMN market_deal_proposals.deal_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.deal_id IS 'Identifier for the deal.';


--
-- Name: COLUMN market_deal_proposals.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.state_root IS 'CID of the parent state root for this deal.';


--
-- Name: COLUMN market_deal_proposals.piece_cid; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.piece_cid IS 'CID of a sector piece. A Piece is an object that represents a whole or part of a File.';


--
-- Name: COLUMN market_deal_proposals.padded_piece_size; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.padded_piece_size IS 'The piece size in bytes with padding.';


--
-- Name: COLUMN market_deal_proposals.unpadded_piece_size; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.unpadded_piece_size IS 'The piece size in bytes without padding.';


--
-- Name: COLUMN market_deal_proposals.is_verified; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.is_verified IS 'Deal is with a verified provider.';


--
-- Name: COLUMN market_deal_proposals.client_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.client_id IS 'Address of the actor proposing the deal.';


--
-- Name: COLUMN market_deal_proposals.provider_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.provider_id IS 'Address of the actor providing the services.';


--
-- Name: COLUMN market_deal_proposals.start_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.start_epoch IS 'The epoch at which this deal with begin. Storage deal must appear in a sealed (proven) sector no later than start_epoch, otherwise it is invalid.';


--
-- Name: COLUMN market_deal_proposals.end_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.end_epoch IS 'The epoch at which this deal with end.';


--
-- Name: COLUMN market_deal_proposals.storage_price_per_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.storage_price_per_epoch IS 'The amount of FIL (in attoFIL) that will be transferred from the client to the provider every epoch this deal is active for.';


--
-- Name: COLUMN market_deal_proposals.provider_collateral; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.provider_collateral IS 'The amount of FIL (in attoFIL) the provider has pledged as collateral. The Provider deal collateral is only slashed when a sector is terminated before the deal expires.';


--
-- Name: COLUMN market_deal_proposals.client_collateral; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.client_collateral IS 'The amount of FIL (in attoFIL) the client has pledged as collateral.';


--
-- Name: COLUMN market_deal_proposals.label; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.label IS 'An arbitrary client chosen label to apply to the deal.';


--
-- Name: COLUMN market_deal_proposals.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_proposals.height IS 'Epoch at which this deal proposal was added or changed.';


--
-- Name: market_deal_states; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.market_deal_states (
    deal_id bigint NOT NULL,
    sector_start_epoch bigint NOT NULL,
    last_update_epoch bigint NOT NULL,
    slash_epoch bigint NOT NULL,
    state_root text NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.market_deal_states OWNER TO postgres;

--
-- Name: TABLE market_deal_states; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.market_deal_states IS 'All storage deal state transitions detected on-chain.';


--
-- Name: COLUMN market_deal_states.deal_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_states.deal_id IS 'Identifier for the deal.';


--
-- Name: COLUMN market_deal_states.sector_start_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_states.sector_start_epoch IS 'Epoch this deal was included in a proven sector. -1 if not yet included in proven sector.';


--
-- Name: COLUMN market_deal_states.last_update_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_states.last_update_epoch IS 'Epoch this deal was last updated at. -1 if deal state never updated.';


--
-- Name: COLUMN market_deal_states.slash_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_states.slash_epoch IS 'Epoch this deal was slashed at. -1 if deal was never slashed.';


--
-- Name: COLUMN market_deal_states.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_states.state_root IS 'CID of the parent state root for this deal.';


--
-- Name: COLUMN market_deal_states.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.market_deal_states.height IS 'Epoch at which this deal was added or changed.';


--
-- Name: message_gas_economy; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.message_gas_economy (
    state_root text NOT NULL,
    gas_limit_total bigint NOT NULL,
    gas_limit_unique_total bigint,
    base_fee double precision NOT NULL,
    base_fee_change_log double precision NOT NULL,
    gas_fill_ratio double precision,
    gas_capacity_ratio double precision,
    gas_waste_ratio double precision,
    height bigint NOT NULL
);


ALTER TABLE public.message_gas_economy OWNER TO postgres;

--
-- Name: TABLE message_gas_economy; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.message_gas_economy IS 'Gas economics for all messages in all blocks at each epoch.';


--
-- Name: COLUMN message_gas_economy.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN message_gas_economy.gas_limit_total; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.gas_limit_total IS 'The sum of all the gas limits.';


--
-- Name: COLUMN message_gas_economy.gas_limit_unique_total; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.gas_limit_unique_total IS 'The sum of all the gas limits of unique messages.';


--
-- Name: COLUMN message_gas_economy.base_fee; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.base_fee IS 'The set price per unit of gas (measured in attoFIL/gas unit) to be burned (sent to an unrecoverable address) for every message execution.';


--
-- Name: COLUMN message_gas_economy.base_fee_change_log; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.base_fee_change_log IS 'The logarithm of the change between new and old base fee.';


--
-- Name: COLUMN message_gas_economy.gas_fill_ratio; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.gas_fill_ratio IS 'The gas_limit_total / target gas limit total for all blocks.';


--
-- Name: COLUMN message_gas_economy.gas_capacity_ratio; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.gas_capacity_ratio IS 'The gas_limit_unique_total / target gas limit total for all blocks.';


--
-- Name: COLUMN message_gas_economy.gas_waste_ratio; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.gas_waste_ratio IS '(gas_limit_total - gas_limit_unique_total) / target gas limit total for all blocks.';


--
-- Name: COLUMN message_gas_economy.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.message_gas_economy.height IS 'Epoch these economics apply to.';


--
-- Name: messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.messages (
    cid text NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    size_bytes bigint NOT NULL,
    nonce bigint NOT NULL,
    value text NOT NULL,
    gas_fee_cap text NOT NULL,
    gas_premium text NOT NULL,
    gas_limit bigint NOT NULL,
    method bigint,
    height bigint NOT NULL
);


ALTER TABLE public.messages OWNER TO postgres;

--
-- Name: TABLE messages; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.messages IS 'Validated on-chain messages by their CID and their metadata.';


--
-- Name: COLUMN messages.cid; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages.cid IS 'CID of the message.';


--
-- Name: COLUMN messages."from"; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages."from" IS 'Address of the actor that sent the message.';


--
-- Name: COLUMN messages."to"; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages."to" IS 'Address of the actor that received the message.';


--
-- Name: COLUMN messages.size_bytes; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages.size_bytes IS 'Size of the serialized message in bytes.';


--
-- Name: COLUMN messages.nonce; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages.nonce IS 'The message nonce, which protects against duplicate messages and multiple messages with the same values.';


--
-- Name: COLUMN messages.value; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages.value IS 'Amount of FIL (in attoFIL) transferred by this message.';


--
-- Name: COLUMN messages.gas_fee_cap; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages.gas_fee_cap IS 'The maximum price that the message sender is willing to pay per unit of gas.';


--
-- Name: COLUMN messages.gas_premium; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages.gas_premium IS 'The price per unit of gas (measured in attoFIL/gas) that the message sender is willing to pay (on top of the BaseFee) to "tip" the miner that will include this message in a block.';


--
-- Name: COLUMN messages.method; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages.method IS 'The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';


--
-- Name: COLUMN messages.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.messages.height IS 'Epoch this message was executed at.';


--
-- Name: miner_current_deadline_infos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_current_deadline_infos (
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


ALTER TABLE public.miner_current_deadline_infos OWNER TO postgres;

--
-- Name: TABLE miner_current_deadline_infos; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_current_deadline_infos IS 'Deadline refers to the window during which proofs may be submitted.';


--
-- Name: COLUMN miner_current_deadline_infos.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.height IS 'Epoch at which this info was calculated.';


--
-- Name: COLUMN miner_current_deadline_infos.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.miner_id IS 'Address of the miner this info relates to.';


--
-- Name: COLUMN miner_current_deadline_infos.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN miner_current_deadline_infos.deadline_index; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.deadline_index IS 'A deadline index, in [0..d.WPoStProvingPeriodDeadlines) unless period elapsed.';


--
-- Name: COLUMN miner_current_deadline_infos.period_start; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.period_start IS 'First epoch of the proving period (<= CurrentEpoch).';


--
-- Name: COLUMN miner_current_deadline_infos.open; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.open IS 'First epoch from which a proof may be submitted (>= CurrentEpoch).';


--
-- Name: COLUMN miner_current_deadline_infos.close; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.close IS 'First epoch from which a proof may no longer be submitted (>= Open).';


--
-- Name: COLUMN miner_current_deadline_infos.challenge; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.challenge IS 'Epoch at which to sample the chain for challenge (< Open).';


--
-- Name: COLUMN miner_current_deadline_infos.fault_cutoff; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_current_deadline_infos.fault_cutoff IS 'First epoch at which a fault declaration is rejected (< Open).';


--
-- Name: miner_fee_debts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_fee_debts (
    height bigint NOT NULL,
    miner_id text NOT NULL,
    state_root text NOT NULL,
    fee_debt text NOT NULL
);


ALTER TABLE public.miner_fee_debts OWNER TO postgres;

--
-- Name: TABLE miner_fee_debts; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_fee_debts IS 'Miner debts per epoch from unpaid fees.';


--
-- Name: COLUMN miner_fee_debts.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_fee_debts.height IS 'Epoch at which this debt applies.';


--
-- Name: COLUMN miner_fee_debts.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_fee_debts.miner_id IS 'Address of the miner that owes fees.';


--
-- Name: COLUMN miner_fee_debts.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_fee_debts.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN miner_fee_debts.fee_debt; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_fee_debts.fee_debt IS 'Absolute value of debt this miner owes from unpaid fees in attoFIL.';


--
-- Name: miner_infos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_infos (
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
    multi_addresses jsonb
);


ALTER TABLE public.miner_infos OWNER TO postgres;

--
-- Name: TABLE miner_infos; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_infos IS 'Miner Account IDs for all associated addresses plus peer ID. See https://docs.filecoin.io/mine/lotus/miner-addresses/ for more information.';


--
-- Name: COLUMN miner_infos.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.height IS 'Epoch at which this miner info was added/changed.';


--
-- Name: COLUMN miner_infos.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.miner_id IS 'Address of miner this info applies to.';


--
-- Name: COLUMN miner_infos.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN miner_infos.owner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.owner_id IS 'Address of actor designated as the owner. The owner address is the address that created the miner, paid the collateral, and has block rewards paid out to it.';


--
-- Name: COLUMN miner_infos.worker_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.worker_id IS 'Address of actor designated as the worker. The worker is responsible for doing all of the work, submitting proofs, committing new sectors, and all other day to day activities.';


--
-- Name: COLUMN miner_infos.new_worker; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.new_worker IS 'Address of a new worker address that will become effective at worker_change_epoch.';


--
-- Name: COLUMN miner_infos.worker_change_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.worker_change_epoch IS 'Epoch at which a new_worker address will become effective.';


--
-- Name: COLUMN miner_infos.consensus_faulted_elapsed; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.consensus_faulted_elapsed IS 'The next epoch this miner is eligible for certain permissioned actor methods and winning block elections as a result of being reported for a consensus fault.';


--
-- Name: COLUMN miner_infos.peer_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.peer_id IS 'Current libp2p Peer ID of the miner.';


--
-- Name: COLUMN miner_infos.control_addresses; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.control_addresses IS 'JSON array of control addresses. Control addresses are used to submit WindowPoSts proofs to the chain. WindowPoSt is the mechanism through which storage is verified in Filecoin and is required by miners to submit proofs for all sectors every 24 hours. Those proofs are submitted as messages to the blockchain and therefore need to pay the respective fees.';


--
-- Name: COLUMN miner_infos.multi_addresses; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_infos.multi_addresses IS 'JSON array of multiaddrs at which this miner can be reached.';


--
-- Name: miner_locked_funds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_locked_funds (
    height bigint NOT NULL,
    miner_id text NOT NULL,
    state_root text NOT NULL,
    locked_funds text NOT NULL,
    initial_pledge text NOT NULL,
    pre_commit_deposits text NOT NULL
);


ALTER TABLE public.miner_locked_funds OWNER TO postgres;

--
-- Name: TABLE miner_locked_funds; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_locked_funds IS 'Details of Miner funds locked and unavailable for use.';


--
-- Name: COLUMN miner_locked_funds.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_locked_funds.height IS 'Epoch at which these details were added/changed.';


--
-- Name: COLUMN miner_locked_funds.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_locked_funds.miner_id IS 'Address of the miner these details apply to.';


--
-- Name: COLUMN miner_locked_funds.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_locked_funds.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN miner_locked_funds.locked_funds; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_locked_funds.locked_funds IS 'Amount of FIL (in attoFIL) locked due to vesting. When a Miner receives tokens from block rewards, the tokens are locked and added to the Miner''s vesting table to be unlocked linearly over some future epochs.';


--
-- Name: COLUMN miner_locked_funds.initial_pledge; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_locked_funds.initial_pledge IS 'Amount of FIL (in attoFIL) locked due to it being pledged as collateral. When a Miner ProveCommits a Sector, they must supply an "initial pledge" for the Sector, which acts as collateral. If the Sector is terminated, this deposit is removed and burned along with rewards earned by this sector up to a limit.';


--
-- Name: COLUMN miner_locked_funds.pre_commit_deposits; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_locked_funds.pre_commit_deposits IS 'Amount of FIL (in attoFIL) locked due to it being used as a PreCommit deposit. When a Miner PreCommits a Sector, they must supply a "precommit deposit" for the Sector, which acts as collateral. If the Sector is not ProveCommitted on time, this deposit is removed and burned.';


--
-- Name: miner_pre_commit_infos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_pre_commit_infos (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    state_root text NOT NULL,
    sealed_cid text NOT NULL,
    seal_rand_epoch bigint,
    expiration_epoch bigint,
    pre_commit_deposit text NOT NULL,
    pre_commit_epoch bigint,
    deal_weight text NOT NULL,
    verified_deal_weight text NOT NULL,
    is_replace_capacity boolean,
    replace_sector_deadline bigint,
    replace_sector_partition bigint,
    replace_sector_number bigint,
    height bigint NOT NULL
);


ALTER TABLE public.miner_pre_commit_infos OWNER TO postgres;

--
-- Name: TABLE miner_pre_commit_infos; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_pre_commit_infos IS 'Information on sector PreCommits.';


--
-- Name: COLUMN miner_pre_commit_infos.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.miner_id IS 'Address of the miner who owns the sector.';


--
-- Name: COLUMN miner_pre_commit_infos.sector_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.sector_id IS 'Numeric identifier for the sector.';


--
-- Name: COLUMN miner_pre_commit_infos.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN miner_pre_commit_infos.sealed_cid; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.sealed_cid IS 'CID of the sealed sector.';


--
-- Name: COLUMN miner_pre_commit_infos.seal_rand_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.seal_rand_epoch IS 'Seal challenge epoch. Epoch at which randomness should be drawn to tie Proof-of-Replication to a chain.';


--
-- Name: COLUMN miner_pre_commit_infos.expiration_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.expiration_epoch IS 'Epoch this sector expires.';


--
-- Name: COLUMN miner_pre_commit_infos.pre_commit_deposit; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.pre_commit_deposit IS 'Amount of FIL (in attoFIL) used as a PreCommit deposit. If the Sector is not ProveCommitted on time, this deposit is removed and burned.';


--
-- Name: COLUMN miner_pre_commit_infos.pre_commit_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.pre_commit_epoch IS 'Epoch this PreCommit was created.';


--
-- Name: COLUMN miner_pre_commit_infos.deal_weight; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.deal_weight IS 'Total space*time of submitted deals.';


--
-- Name: COLUMN miner_pre_commit_infos.verified_deal_weight; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.verified_deal_weight IS 'Total space*time of submitted verified deals.';


--
-- Name: COLUMN miner_pre_commit_infos.is_replace_capacity; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.is_replace_capacity IS 'Whether to replace a "committed capacity" no-deal sector (requires non-empty DealIDs).';


--
-- Name: COLUMN miner_pre_commit_infos.replace_sector_deadline; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.replace_sector_deadline IS 'The deadline location of the sector to replace.';


--
-- Name: COLUMN miner_pre_commit_infos.replace_sector_partition; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.replace_sector_partition IS 'The partition location of the sector to replace.';


--
-- Name: COLUMN miner_pre_commit_infos.replace_sector_number; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.replace_sector_number IS 'ID of the committed capacity sector to replace.';


--
-- Name: COLUMN miner_pre_commit_infos.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_pre_commit_infos.height IS 'Epoch this PreCommit information was added/changed.';


--
-- Name: miner_sector_deals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_sector_deals (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    deal_id bigint NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.miner_sector_deals OWNER TO postgres;

--
-- Name: TABLE miner_sector_deals; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_sector_deals IS 'Mapping of Deal IDs to their respective Miner and Sector IDs.';


--
-- Name: COLUMN miner_sector_deals.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_deals.miner_id IS 'Address of the miner the deal is with.';


--
-- Name: COLUMN miner_sector_deals.sector_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_deals.sector_id IS 'Numeric identifier of the sector the deal is for.';


--
-- Name: COLUMN miner_sector_deals.deal_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_deals.deal_id IS 'Numeric identifier for the deal.';


--
-- Name: COLUMN miner_sector_deals.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_deals.height IS 'Epoch at which this deal was added/updated.';


--
-- Name: miner_sector_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_sector_events (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    state_root text NOT NULL,
    event public.miner_sector_event_type NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.miner_sector_events OWNER TO postgres;

--
-- Name: TABLE miner_sector_events; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_sector_events IS 'Sector events on-chain per Miner/Sector.';


--
-- Name: COLUMN miner_sector_events.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_events.miner_id IS 'Address of the miner who owns the sector.';


--
-- Name: COLUMN miner_sector_events.sector_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_events.sector_id IS 'Numeric identifier of the sector.';


--
-- Name: COLUMN miner_sector_events.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_events.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN miner_sector_events.event; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_events.event IS 'Name of the event that occurred.';


--
-- Name: COLUMN miner_sector_events.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_events.height IS 'Epoch at which this event occurred.';


--
-- Name: miner_sector_infos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_sector_infos (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    state_root text NOT NULL,
    sealed_cid text NOT NULL,
    activation_epoch bigint,
    expiration_epoch bigint,
    deal_weight text NOT NULL,
    verified_deal_weight text NOT NULL,
    initial_pledge text NOT NULL,
    expected_day_reward text NOT NULL,
    expected_storage_pledge text NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.miner_sector_infos OWNER TO postgres;

--
-- Name: TABLE miner_sector_infos; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_sector_infos IS 'Latest state of sectors by Miner.';


--
-- Name: COLUMN miner_sector_infos.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.miner_id IS 'Address of the miner who owns the sector.';


--
-- Name: COLUMN miner_sector_infos.sector_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.sector_id IS 'Numeric identifier of the sector.';


--
-- Name: COLUMN miner_sector_infos.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN miner_sector_infos.sealed_cid; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.sealed_cid IS 'The root CID of the Sealed Sector’s merkle tree. Also called CommR, or "replica commitment".';


--
-- Name: COLUMN miner_sector_infos.activation_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.activation_epoch IS 'Epoch during which the sector proof was accepted.';


--
-- Name: COLUMN miner_sector_infos.expiration_epoch; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.expiration_epoch IS 'Epoch during which the sector expires.';


--
-- Name: COLUMN miner_sector_infos.deal_weight; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.deal_weight IS 'Integral of active deals over sector lifetime.';


--
-- Name: COLUMN miner_sector_infos.verified_deal_weight; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.verified_deal_weight IS 'Integral of active verified deals over sector lifetime.';


--
-- Name: COLUMN miner_sector_infos.initial_pledge; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.initial_pledge IS 'Pledge collected to commit this sector (in attoFIL).';


--
-- Name: COLUMN miner_sector_infos.expected_day_reward; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.expected_day_reward IS 'Expected one day projection of reward for sector computed at activation time (in attoFIL).';


--
-- Name: COLUMN miner_sector_infos.expected_storage_pledge; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.expected_storage_pledge IS 'Expected twenty day projection of reward for sector computed at activation time (in attoFIL).';


--
-- Name: COLUMN miner_sector_infos.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_infos.height IS 'Epoch at which this sector info was added/updated.';


--
-- Name: miner_sector_posts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_sector_posts (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    height bigint NOT NULL,
    post_message_cid text
);


ALTER TABLE public.miner_sector_posts OWNER TO postgres;

--
-- Name: TABLE miner_sector_posts; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.miner_sector_posts IS 'Proof of Spacetime for sectors.';


--
-- Name: COLUMN miner_sector_posts.miner_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_posts.miner_id IS 'Address of the miner who owns the sector.';


--
-- Name: COLUMN miner_sector_posts.sector_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_posts.sector_id IS 'Numeric identifier of the sector.';


--
-- Name: COLUMN miner_sector_posts.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_posts.height IS 'Epoch at which this PoSt message was executed.';


--
-- Name: COLUMN miner_sector_posts.post_message_cid; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.miner_sector_posts.post_message_cid IS 'CID of the PoSt message.';


--
-- Name: multisig_approvals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.multisig_approvals (
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


ALTER TABLE public.multisig_approvals OWNER TO postgres;

--
-- Name: multisig_transactions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.multisig_transactions (
    height bigint NOT NULL,
    multisig_id text NOT NULL,
    state_root text NOT NULL,
    transaction_id bigint NOT NULL,
    "to" text NOT NULL,
    value text NOT NULL,
    method bigint NOT NULL,
    params bytea NOT NULL,
    approved jsonb NOT NULL
);


ALTER TABLE public.multisig_transactions OWNER TO postgres;

--
-- Name: TABLE multisig_transactions; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.multisig_transactions IS 'Details of pending transactions involving multisig actors.';


--
-- Name: COLUMN multisig_transactions.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions.height IS 'Epoch at which this transaction was executed.';


--
-- Name: COLUMN multisig_transactions.multisig_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions.multisig_id IS 'Address of the multisig actor involved in the transaction.';


--
-- Name: COLUMN multisig_transactions.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions.state_root IS 'CID of the parent state root at this epoch.';


--
-- Name: COLUMN multisig_transactions.transaction_id; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions.transaction_id IS 'Number identifier for the transaction - unique per multisig.';


--
-- Name: COLUMN multisig_transactions."to"; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions."to" IS 'Address of the recipient who will be sent a message if the proposal is approved.';


--
-- Name: COLUMN multisig_transactions.value; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions.value IS 'Amount of FIL (in attoFIL) that will be transferred if the proposal is approved.';


--
-- Name: COLUMN multisig_transactions.method; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions.method IS 'The method number to invoke on the recipient if the proposal is approved. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';


--
-- Name: COLUMN multisig_transactions.params; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions.params IS 'CBOR encoded bytes of parameters to send to the method that will be invoked if the proposal is approved.';


--
-- Name: COLUMN multisig_transactions.approved; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.multisig_transactions.approved IS 'Addresses of signers who have approved the transaction. 0th entry is the proposer.';


--
-- Name: parsed_messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.parsed_messages (
    cid text NOT NULL,
    height bigint NOT NULL,
    "from" text NOT NULL,
    "to" text NOT NULL,
    value text NOT NULL,
    method text NOT NULL,
    params jsonb
);


ALTER TABLE public.parsed_messages OWNER TO postgres;

--
-- Name: TABLE parsed_messages; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.parsed_messages IS 'Messages parsed to extract useful information.';


--
-- Name: COLUMN parsed_messages.cid; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.parsed_messages.cid IS 'CID of the message.';


--
-- Name: COLUMN parsed_messages.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.parsed_messages.height IS 'Epoch this message was executed at.';


--
-- Name: COLUMN parsed_messages."from"; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.parsed_messages."from" IS 'Address of the actor that sent the message.';


--
-- Name: COLUMN parsed_messages."to"; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.parsed_messages."to" IS 'Address of the actor that received the message.';


--
-- Name: COLUMN parsed_messages.value; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.parsed_messages.value IS 'Amount of FIL (in attoFIL) transferred by this message.';


--
-- Name: COLUMN parsed_messages.method; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.parsed_messages.method IS 'The name of the method that was invoked on the recipient actor.';


--
-- Name: COLUMN parsed_messages.params; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.parsed_messages.params IS 'Method paramaters parsed and serialized as a JSON object.';


--
-- Name: receipts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.receipts (
    message text NOT NULL,
    state_root text NOT NULL,
    idx bigint NOT NULL,
    exit_code bigint NOT NULL,
    gas_used bigint NOT NULL,
    height bigint NOT NULL
);


ALTER TABLE public.receipts OWNER TO postgres;

--
-- Name: TABLE receipts; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.receipts IS 'Message reciepts after being applied to chain state by message CID and parent state root CID of tipset when message was executed.';


--
-- Name: COLUMN receipts.message; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.receipts.message IS 'CID of the message this receipt belongs to.';


--
-- Name: COLUMN receipts.state_root; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.receipts.state_root IS 'CID of the parent state root that this epoch.';


--
-- Name: COLUMN receipts.idx; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.receipts.idx IS 'Index of message indicating execution order.';


--
-- Name: COLUMN receipts.exit_code; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.receipts.exit_code IS 'The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific.';


--
-- Name: COLUMN receipts.gas_used; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.receipts.gas_used IS 'A measure of the amount of resources (or units of gas) consumed, in order to execute a message.';


--
-- Name: COLUMN receipts.height; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON COLUMN public.receipts.height IS 'Epoch the message was executed and receipt generated.';


--
-- Name: sector_precommit_info; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.sector_precommit_info (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    sealed_cid text NOT NULL,
    state_root text NOT NULL,
    seal_rand_epoch bigint NOT NULL,
    expiration_epoch bigint NOT NULL,
    precommit_deposit text NOT NULL,
    precommit_epoch bigint NOT NULL,
    deal_weight text NOT NULL,
    verified_deal_weight text NOT NULL,
    is_replace_capacity boolean NOT NULL,
    replace_sector_deadline bigint,
    replace_sector_partition bigint,
    replace_sector_number bigint
);


ALTER TABLE public.sector_precommit_info OWNER TO postgres;

--
-- Name: state_heights; Type: MATERIALIZED VIEW; Schema: public; Owner: postgres
--

CREATE MATERIALIZED VIEW public.state_heights AS
 SELECT DISTINCT block_headers.height,
    block_headers.parent_state_root AS parentstateroot
   FROM public.block_headers
  WITH NO DATA;


ALTER TABLE public.state_heights OWNER TO postgres;

--
-- Name: visor_processing_reports; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.visor_processing_reports (
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


ALTER TABLE public.visor_processing_reports OWNER TO postgres;

--
-- Name: visor_version; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.visor_version (
    major integer NOT NULL
);


ALTER TABLE public.visor_version OWNER TO postgres;

--
-- Name: gopg_migrations id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.gopg_migrations ALTER COLUMN id SET DEFAULT nextval('public.gopg_migrations_id_seq'::regclass);


--
-- Name: actor_states actor_states_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.actor_states
    ADD CONSTRAINT actor_states_pkey PRIMARY KEY (height, head, code);


--
-- Name: actors actors_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.actors
    ADD CONSTRAINT actors_pkey PRIMARY KEY (height, id, state_root);


--
-- Name: block_headers block_headers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_headers
    ADD CONSTRAINT block_headers_pkey PRIMARY KEY (height, cid);


--
-- Name: block_messages block_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_messages
    ADD CONSTRAINT block_messages_pkey PRIMARY KEY (height, block, message);


--
-- Name: block_parents block_parents_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_parents
    ADD CONSTRAINT block_parents_pkey PRIMARY KEY (height, block, parent);


--
-- Name: blocks_synced blocks_synced_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks_synced
    ADD CONSTRAINT blocks_synced_pk PRIMARY KEY (cid);


--
-- Name: chain_economics chain_economics_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chain_economics
    ADD CONSTRAINT chain_economics_pk PRIMARY KEY (parent_state_root);


--
-- Name: chain_powers chain_powers_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chain_powers
    ADD CONSTRAINT chain_powers_pkey PRIMARY KEY (height, state_root);


--
-- Name: chain_rewards chain_rewards_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chain_rewards
    ADD CONSTRAINT chain_rewards_pkey PRIMARY KEY (height, state_root);


--
-- Name: derived_gas_outputs derived_gas_outputs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.derived_gas_outputs
    ADD CONSTRAINT derived_gas_outputs_pkey PRIMARY KEY (height, cid, state_root);


--
-- Name: id_address_map id_address_map_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.id_address_map
    ADD CONSTRAINT id_address_map_pk PRIMARY KEY (id, address);


--
-- Name: id_addresses id_addresses_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.id_addresses
    ADD CONSTRAINT id_addresses_pkey PRIMARY KEY (id, address, state_root);


--
-- Name: market_deal_proposals market_deal_proposals_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.market_deal_proposals
    ADD CONSTRAINT market_deal_proposals_pkey PRIMARY KEY (height, deal_id);


--
-- Name: market_deal_states market_deal_states_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.market_deal_states
    ADD CONSTRAINT market_deal_states_pkey PRIMARY KEY (height, deal_id, state_root);


--
-- Name: message_gas_economy message_gas_economy_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.message_gas_economy
    ADD CONSTRAINT message_gas_economy_pkey PRIMARY KEY (height, state_root);


--
-- Name: messages messages_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_pkey PRIMARY KEY (height, cid);


--
-- Name: miner_current_deadline_infos miner_current_deadline_infos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_current_deadline_infos
    ADD CONSTRAINT miner_current_deadline_infos_pkey PRIMARY KEY (height, miner_id, state_root);


--
-- Name: miner_fee_debts miner_fee_debts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_fee_debts
    ADD CONSTRAINT miner_fee_debts_pkey PRIMARY KEY (height, miner_id, state_root);


--
-- Name: miner_infos miner_infos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_infos
    ADD CONSTRAINT miner_infos_pkey PRIMARY KEY (height, miner_id, state_root);


--
-- Name: miner_locked_funds miner_locked_funds_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_locked_funds
    ADD CONSTRAINT miner_locked_funds_pkey PRIMARY KEY (height, miner_id, state_root);


--
-- Name: miner_pre_commit_infos miner_pre_commit_infos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_pre_commit_infos
    ADD CONSTRAINT miner_pre_commit_infos_pkey PRIMARY KEY (height, miner_id, sector_id, state_root);


--
-- Name: miner_sector_deals miner_sector_deals_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_sector_deals
    ADD CONSTRAINT miner_sector_deals_pkey PRIMARY KEY (height, miner_id, sector_id, deal_id);


--
-- Name: miner_sector_events miner_sector_events_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_sector_events
    ADD CONSTRAINT miner_sector_events_pkey PRIMARY KEY (height, sector_id, event, miner_id, state_root);


--
-- Name: miner_sector_infos miner_sector_infos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_sector_infos
    ADD CONSTRAINT miner_sector_infos_pkey PRIMARY KEY (height, miner_id, sector_id, state_root);


--
-- Name: miner_sector_posts miner_sector_posts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_sector_posts
    ADD CONSTRAINT miner_sector_posts_pkey PRIMARY KEY (height, miner_id, sector_id);


--
-- Name: multisig_approvals multisig_approvals_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.multisig_approvals
    ADD CONSTRAINT multisig_approvals_pkey PRIMARY KEY (height, state_root, multisig_id, message, approver);


--
-- Name: multisig_transactions multisig_transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.multisig_transactions
    ADD CONSTRAINT multisig_transactions_pkey PRIMARY KEY (height, state_root, multisig_id, transaction_id);


--
-- Name: parsed_messages parsed_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.parsed_messages
    ADD CONSTRAINT parsed_messages_pkey PRIMARY KEY (height, cid);


--
-- Name: power_actor_claims power_actor_claims_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.power_actor_claims
    ADD CONSTRAINT power_actor_claims_pkey PRIMARY KEY (height, miner_id, state_root);


--
-- Name: receipts receipts_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.receipts
    ADD CONSTRAINT receipts_pkey PRIMARY KEY (height, message, state_root);


--
-- Name: sector_precommit_info sector_precommit_info_miner_id_sector_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sector_precommit_info
    ADD CONSTRAINT sector_precommit_info_miner_id_sector_id_key UNIQUE (miner_id, sector_id);


--
-- Name: sector_precommit_info sector_precommit_info_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sector_precommit_info
    ADD CONSTRAINT sector_precommit_info_pk PRIMARY KEY (miner_id, sector_id, sealed_cid);


--
-- Name: visor_processing_reports visor_processing_reports_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.visor_processing_reports
    ADD CONSTRAINT visor_processing_reports_pkey PRIMARY KEY (height, state_root, reporter, task, started_at);


--
-- Name: visor_version visor_version_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.visor_version
    ADD CONSTRAINT visor_version_pkey PRIMARY KEY (major);


--
-- Name: actor_states_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX actor_states_height_idx ON public.actor_states USING btree (height DESC);


--
-- Name: actors_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX actors_height_idx ON public.actors USING btree (height DESC);


--
-- Name: block_drand_entries_round_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX block_drand_entries_round_uindex ON public.drand_block_entries USING btree (round, block);


--
-- Name: block_headers_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX block_headers_height_idx ON public.block_headers USING btree (height DESC);


--
-- Name: block_headers_timestamp_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX block_headers_timestamp_idx ON public.block_headers USING btree ("timestamp");


--
-- Name: block_messages_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX block_messages_height_idx ON public.block_messages USING btree (height DESC);


--
-- Name: block_parents_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX block_parents_height_idx ON public.block_parents USING btree (height DESC);


--
-- Name: blocks_synced_cid_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX blocks_synced_cid_uindex ON public.blocks_synced USING btree (cid, processed_at);


--
-- Name: chain_powers_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX chain_powers_height_idx ON public.chain_powers USING btree (height DESC);


--
-- Name: chain_rewards_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX chain_rewards_height_idx ON public.chain_rewards USING btree (height DESC);


--
-- Name: derived_gas_outputs_exit_code_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX derived_gas_outputs_exit_code_index ON public.derived_gas_outputs USING btree (exit_code);


--
-- Name: derived_gas_outputs_from_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX derived_gas_outputs_from_index ON public.derived_gas_outputs USING hash ("from");


--
-- Name: derived_gas_outputs_method_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX derived_gas_outputs_method_index ON public.derived_gas_outputs USING btree (method);


--
-- Name: derived_gas_outputs_to_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX derived_gas_outputs_to_index ON public.derived_gas_outputs USING hash ("to");


--
-- Name: id_address_map_address_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX id_address_map_address_index ON public.id_address_map USING btree (address);


--
-- Name: id_address_map_address_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX id_address_map_address_uindex ON public.id_address_map USING btree (address);


--
-- Name: id_address_map_id_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX id_address_map_id_index ON public.id_address_map USING btree (id);


--
-- Name: id_address_map_id_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX id_address_map_id_uindex ON public.id_address_map USING btree (id);


--
-- Name: market_deal_proposals_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX market_deal_proposals_height_idx ON public.market_deal_proposals USING btree (height DESC);


--
-- Name: market_deal_states_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX market_deal_states_height_idx ON public.market_deal_states USING btree (height DESC);


--
-- Name: message_gas_economy_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX message_gas_economy_height_idx ON public.message_gas_economy USING btree (height DESC);


--
-- Name: message_parsed_from_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX message_parsed_from_idx ON public.parsed_messages USING hash ("from");


--
-- Name: message_parsed_method_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX message_parsed_method_idx ON public.parsed_messages USING hash (method);


--
-- Name: message_parsed_to_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX message_parsed_to_idx ON public.parsed_messages USING hash ("to");


--
-- Name: messages_from_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX messages_from_index ON public.messages USING btree ("from");


--
-- Name: messages_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX messages_height_idx ON public.messages USING btree (height DESC);


--
-- Name: messages_to_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX messages_to_index ON public.messages USING btree ("to");


--
-- Name: miner_current_deadline_infos_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_current_deadline_infos_height_idx ON public.miner_current_deadline_infos USING btree (height DESC);


--
-- Name: miner_deal_sectors_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_deal_sectors_height_idx ON public.miner_sector_deals USING btree (height DESC);


--
-- Name: miner_fee_debts_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_fee_debts_height_idx ON public.miner_fee_debts USING btree (height DESC);


--
-- Name: miner_infos_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_infos_height_idx ON public.miner_infos USING btree (height DESC);


--
-- Name: miner_locked_funds_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_locked_funds_height_idx ON public.miner_locked_funds USING btree (height DESC);


--
-- Name: miner_pre_commit_infos_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_pre_commit_infos_height_idx ON public.miner_pre_commit_infos USING btree (height DESC);


--
-- Name: miner_sector_events_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_sector_events_height_idx ON public.miner_sector_events USING btree (height DESC);


--
-- Name: miner_sector_infos_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_sector_infos_height_idx ON public.miner_sector_infos USING btree (height DESC);


--
-- Name: miner_sector_posts_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX miner_sector_posts_height_idx ON public.miner_sector_posts USING btree (height DESC);


--
-- Name: multisig_approvals_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX multisig_approvals_height_idx ON public.multisig_approvals USING btree (height DESC);


--
-- Name: parsed_messages_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX parsed_messages_height_idx ON public.parsed_messages USING btree (height DESC);


--
-- Name: receipts_height_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX receipts_height_idx ON public.receipts USING btree (height DESC);


--
-- Name: state_heights_height_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX state_heights_height_index ON public.state_heights USING btree (height);


--
-- Name: state_heights_parentstateroot_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX state_heights_parentstateroot_index ON public.state_heights USING btree (parentstateroot);


--
-- Name: actor_states ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.actor_states FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: actors ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.actors FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: block_headers ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.block_headers FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: block_messages ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.block_messages FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: block_parents ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.block_parents FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: chain_powers ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.chain_powers FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: chain_rewards ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.chain_rewards FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: market_deal_proposals ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.market_deal_proposals FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: market_deal_states ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.market_deal_states FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: message_gas_economy ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.message_gas_economy FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: messages ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.messages FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_current_deadline_infos ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_current_deadline_infos FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_fee_debts ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_fee_debts FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_infos ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_infos FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_locked_funds ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_locked_funds FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_pre_commit_infos ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_pre_commit_infos FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_sector_deals ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_sector_deals FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_sector_events ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_sector_events FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_sector_infos ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_sector_infos FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: miner_sector_posts ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.miner_sector_posts FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: multisig_approvals ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.multisig_approvals FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: parsed_messages ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.parsed_messages FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- Name: receipts ts_insert_blocker; Type: TRIGGER; Schema: public; Owner: postgres
--

CREATE TRIGGER ts_insert_blocker BEFORE INSERT ON public.receipts FOR EACH ROW EXECUTE FUNCTION _timescaledb_internal.insert_blocker();


--
-- PostgreSQL database dump complete
--

`
