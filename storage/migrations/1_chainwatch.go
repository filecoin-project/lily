package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 1 is the schema used by chainwatch

func init() {

	up := batch(`
--
-- Name: timescaledb; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS timescaledb WITH SCHEMA public;

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


SET default_tablespace = '';

--
-- Name: actor_states; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.actor_states (
    head text NOT NULL,
    code text NOT NULL,
    state json NOT NULL
);


ALTER TABLE public.actor_states OWNER TO postgres;

--
-- Name: actors; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.actors (
    id text NOT NULL,
    code text NOT NULL,
    head text NOT NULL,
    nonce integer NOT NULL,
    balance text NOT NULL,
    stateroot text
);


ALTER TABLE public.actors OWNER TO postgres;

--
-- Name: block_cids; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.block_cids (
    cid text NOT NULL
);


ALTER TABLE public.block_cids OWNER TO postgres;

--
-- Name: block_drand_entries; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.block_drand_entries (
    round bigint NOT NULL,
    block text NOT NULL
);


ALTER TABLE public.block_drand_entries OWNER TO postgres;

--
-- Name: block_messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.block_messages (
    block text NOT NULL,
    message text NOT NULL
);


ALTER TABLE public.block_messages OWNER TO postgres;

--
-- Name: block_parents; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.block_parents (
    block text NOT NULL,
    parent text NOT NULL
);


ALTER TABLE public.block_parents OWNER TO postgres;

--
-- Name: blocks; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.blocks (
    cid text NOT NULL,
    parentweight numeric NOT NULL,
    parentstateroot text NOT NULL,
    height bigint NOT NULL,
    miner text NOT NULL,
    "timestamp" bigint NOT NULL,
    ticket bytea NOT NULL,
    election_proof bytea,
    win_count bigint,
    parent_base_fee text NOT NULL,
    forksig bigint NOT NULL
);


ALTER TABLE public.blocks OWNER TO postgres;

--
-- Name: blocks_synced; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.blocks_synced (
    cid text NOT NULL,
    synced_at integer NOT NULL,
    processed_at integer
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
-- Name: chain_power; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chain_power (
    state_root text NOT NULL,
    new_raw_bytes_power text NOT NULL,
    new_qa_bytes_power text NOT NULL,
    new_pledge_collateral text NOT NULL,
    total_raw_bytes_power text NOT NULL,
    total_raw_bytes_committed text NOT NULL,
    total_qa_bytes_power text NOT NULL,
    total_qa_bytes_committed text NOT NULL,
    total_pledge_collateral text NOT NULL,
    qa_smoothed_position_estimate text NOT NULL,
    qa_smoothed_velocity_estimate text NOT NULL,
    miner_count integer NOT NULL,
    minimum_consensus_miner_count integer NOT NULL
);


ALTER TABLE public.chain_power OWNER TO postgres;

--
-- Name: chain_reward; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.chain_reward (
    state_root text NOT NULL,
    cum_sum_baseline text NOT NULL,
    cum_sum_realized text NOT NULL,
    effective_network_time integer NOT NULL,
    effective_baseline_power text NOT NULL,
    new_baseline_power text NOT NULL,
    new_reward numeric NOT NULL,
    new_reward_smoothed_position_estimate text NOT NULL,
    new_reward_smoothed_velocity_estimate text NOT NULL,
    total_mined_reward text NOT NULL
);


ALTER TABLE public.chain_reward OWNER TO postgres;

--
-- Name: drand_entries; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.drand_entries (
    round bigint NOT NULL,
    data bytea NOT NULL
);


ALTER TABLE public.drand_entries OWNER TO postgres;

--
-- Name: id_address_map; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.id_address_map (
    id text NOT NULL,
    address text NOT NULL
);


ALTER TABLE public.id_address_map OWNER TO postgres;

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
    client_collateral text NOT NULL
);


ALTER TABLE public.market_deal_proposals OWNER TO postgres;

--
-- Name: market_deal_states; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.market_deal_states (
    deal_id bigint NOT NULL,
    sector_start_epoch bigint NOT NULL,
    last_update_epoch bigint NOT NULL,
    slash_epoch bigint NOT NULL,
    state_root text NOT NULL
);


ALTER TABLE public.market_deal_states OWNER TO postgres;

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
    params bytea
);


ALTER TABLE public.messages OWNER TO postgres;

--
-- Name: miner_info; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_info (
    miner_id text NOT NULL,
    owner_addr text NOT NULL,
    worker_addr text NOT NULL,
    peer_id text,
    sector_size text NOT NULL
);


ALTER TABLE public.miner_info OWNER TO postgres;

--
-- Name: miner_power; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_power (
    miner_id text NOT NULL,
    state_root text NOT NULL,
    raw_bytes_power text NOT NULL,
    quality_adjusted_power text NOT NULL
);


ALTER TABLE public.miner_power OWNER TO postgres;

--
-- Name: miner_sector_events; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.miner_sector_events (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    state_root text NOT NULL,
    event public.miner_sector_event_type NOT NULL
);


ALTER TABLE public.miner_sector_events OWNER TO postgres;

--
-- Name: minerid_dealid_sectorid; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.minerid_dealid_sectorid (
    deal_id bigint NOT NULL,
    sector_id bigint NOT NULL,
    miner_id text NOT NULL
);


ALTER TABLE public.minerid_dealid_sectorid OWNER TO postgres;

--
-- Name: mpool_messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.mpool_messages (
    msg text NOT NULL,
    add_ts integer NOT NULL
);


ALTER TABLE public.mpool_messages OWNER TO postgres;

--
-- Name: receipts; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.receipts (
    msg text NOT NULL,
    state text NOT NULL,
    idx integer NOT NULL,
    exit integer NOT NULL,
    gas_used bigint NOT NULL,
    return bytea
);


ALTER TABLE public.receipts OWNER TO postgres;

--
-- Name: sector_info; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.sector_info (
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
    sealed_cid text NOT NULL,
    state_root text NOT NULL,
    activation_epoch bigint NOT NULL,
    expiration_epoch bigint NOT NULL,
    deal_weight text NOT NULL,
    verified_deal_weight text NOT NULL,
    initial_pledge text NOT NULL,
    expected_day_reward text NOT NULL,
    expected_storage_pledge text NOT NULL
);


ALTER TABLE public.sector_info OWNER TO postgres;

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
 SELECT DISTINCT blocks.height,
    blocks.parentstateroot
   FROM public.blocks
  WITH NO DATA;


ALTER TABLE public.state_heights OWNER TO postgres;

--
-- Name: top_miners_by_base_reward; Type: MATERIALIZED VIEW; Schema: public; Owner: postgres
--

CREATE MATERIALIZED VIEW public.top_miners_by_base_reward AS
 WITH total_rewards_by_miner AS (
         SELECT b.miner,
            sum((cr.new_reward * (b.win_count)::numeric)) AS total_reward
           FROM (public.blocks b
             JOIN public.chain_reward cr ON ((b.parentstateroot = cr.state_root)))
          GROUP BY b.miner
        )
 SELECT rank() OVER (ORDER BY total_rewards_by_miner.total_reward DESC) AS rank,
    total_rewards_by_miner.miner,
    total_rewards_by_miner.total_reward
   FROM total_rewards_by_miner
  GROUP BY total_rewards_by_miner.miner, total_rewards_by_miner.total_reward
  WITH NO DATA;


ALTER TABLE public.top_miners_by_base_reward OWNER TO postgres;

--
-- Name: top_miners_by_base_reward_max_height; Type: MATERIALIZED VIEW; Schema: public; Owner: postgres
--

CREATE MATERIALIZED VIEW public.top_miners_by_base_reward_max_height AS
 SELECT b."timestamp" AS "current_timestamp",
    max(b.height) AS current_height
   FROM (public.blocks b
     JOIN public.chain_reward cr ON ((b.parentstateroot = cr.state_root)))
  WHERE (cr.new_reward IS NOT NULL)
  GROUP BY b."timestamp"
  ORDER BY b."timestamp" DESC
 LIMIT 1
  WITH NO DATA;


ALTER TABLE public.top_miners_by_base_reward_max_height OWNER TO postgres;

--
-- Name: block_cids block_cids_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_cids
    ADD CONSTRAINT block_cids_pk PRIMARY KEY (cid);


--
-- Name: block_messages block_messages_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_messages
    ADD CONSTRAINT block_messages_pk PRIMARY KEY (block, message);


--
-- Name: blocks blocks_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_pk PRIMARY KEY (cid);


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
-- Name: chain_reward chain_reward_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chain_reward
    ADD CONSTRAINT chain_reward_pk PRIMARY KEY (state_root);


--
-- Name: drand_entries drand_entries_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.drand_entries
    ADD CONSTRAINT drand_entries_pk PRIMARY KEY (round);


--
-- Name: id_address_map id_address_map_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.id_address_map
    ADD CONSTRAINT id_address_map_pk PRIMARY KEY (id, address);


--
-- Name: market_deal_proposals market_deal_proposal_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.market_deal_proposals
    ADD CONSTRAINT market_deal_proposal_pk PRIMARY KEY (deal_id);


--
-- Name: market_deal_states market_deal_states_deal_id_sector_start_epoch_last_update_e_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.market_deal_states
    ADD CONSTRAINT market_deal_states_deal_id_sector_start_epoch_last_update_e_key UNIQUE (deal_id, sector_start_epoch, last_update_epoch, slash_epoch);


--
-- Name: market_deal_states market_deal_states_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.market_deal_states
    ADD CONSTRAINT market_deal_states_pk PRIMARY KEY (deal_id, state_root);


--
-- Name: messages messages_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_pk PRIMARY KEY (cid);


--
-- Name: miner_info miner_info_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_info
    ADD CONSTRAINT miner_info_pk PRIMARY KEY (miner_id);


--
-- Name: miner_power miner_power_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_power
    ADD CONSTRAINT miner_power_pk PRIMARY KEY (miner_id, state_root);


--
-- Name: minerid_dealid_sectorid miner_sector_deal_ids_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.minerid_dealid_sectorid
    ADD CONSTRAINT miner_sector_deal_ids_pk PRIMARY KEY (miner_id, sector_id, deal_id);


--
-- Name: miner_sector_events miner_sector_events_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.miner_sector_events
    ADD CONSTRAINT miner_sector_events_pk PRIMARY KEY (sector_id, event, miner_id, state_root);


--
-- Name: mpool_messages mpool_messages_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.mpool_messages
    ADD CONSTRAINT mpool_messages_pk PRIMARY KEY (msg);


--
-- Name: chain_power power_smoothing_estimates_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.chain_power
    ADD CONSTRAINT power_smoothing_estimates_pk PRIMARY KEY (state_root);


--
-- Name: receipts receipts_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.receipts
    ADD CONSTRAINT receipts_pk PRIMARY KEY (msg, state);


--
-- Name: sector_info sector_info_pk; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.sector_info
    ADD CONSTRAINT sector_info_pk PRIMARY KEY (miner_id, sector_id, sealed_cid);


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
-- Name: actor_states_code_head_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX actor_states_code_head_index ON public.actor_states USING btree (head, code);


--
-- Name: actor_states_head_code_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX actor_states_head_code_uindex ON public.actor_states USING btree (head, code);


--
-- Name: actor_states_head_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX actor_states_head_index ON public.actor_states USING btree (head);


--
-- Name: actors_id_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX actors_id_index ON public.actors USING btree (id);


--
-- Name: block_cid_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX block_cid_uindex ON public.blocks USING btree (cid, height);


--
-- Name: block_cids_cid_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX block_cids_cid_uindex ON public.block_cids USING btree (cid);


--
-- Name: block_drand_entries_round_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX block_drand_entries_round_uindex ON public.block_drand_entries USING btree (round, block);


--
-- Name: block_parents_block_parent_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX block_parents_block_parent_uindex ON public.block_parents USING btree (block, parent);


--
-- Name: blocks_synced_cid_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX blocks_synced_cid_uindex ON public.blocks_synced USING btree (cid, processed_at);


--
-- Name: drand_entries_round_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX drand_entries_round_uindex ON public.drand_entries USING btree (round);


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
-- Name: messages_cid_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX messages_cid_uindex ON public.messages USING btree (cid);


--
-- Name: messages_from_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX messages_from_index ON public.messages USING btree ("from");


--
-- Name: messages_to_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX messages_to_index ON public.messages USING btree ("to");


--
-- Name: mpool_messages_msg_uindex; Type: INDEX; Schema: public; Owner: postgres
--

CREATE UNIQUE INDEX mpool_messages_msg_uindex ON public.mpool_messages USING btree (msg);


--
-- Name: receipts_msg_state_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX receipts_msg_state_index ON public.receipts USING btree (msg, state);


--
-- Name: state_heights_height_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX state_heights_height_index ON public.state_heights USING btree (height);


--
-- Name: state_heights_parentstateroot_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX state_heights_parentstateroot_index ON public.state_heights USING btree (parentstateroot);


--
-- Name: top_miners_by_base_reward_miner_index; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX top_miners_by_base_reward_miner_index ON public.top_miners_by_base_reward USING btree (miner);


--
-- Name: block_drand_entries block_drand_entries_drand_entries_round_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_drand_entries
    ADD CONSTRAINT block_drand_entries_drand_entries_round_fk FOREIGN KEY (round) REFERENCES public.drand_entries(round);


--
-- Name: blocks_synced blocks_block_cids_cid_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks_synced
    ADD CONSTRAINT blocks_block_cids_cid_fk FOREIGN KEY (cid) REFERENCES public.block_cids(cid);


--
-- Name: block_parents blocks_block_cids_cid_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_parents
    ADD CONSTRAINT blocks_block_cids_cid_fk FOREIGN KEY (block) REFERENCES public.block_cids(cid);


--
-- Name: block_drand_entries blocks_block_cids_cid_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_drand_entries
    ADD CONSTRAINT blocks_block_cids_cid_fk FOREIGN KEY (block) REFERENCES public.block_cids(cid);


--
-- Name: blocks blocks_block_cids_cid_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_block_cids_cid_fk FOREIGN KEY (cid) REFERENCES public.block_cids(cid);


--
-- Name: block_messages blocks_block_cids_cid_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.block_messages
    ADD CONSTRAINT blocks_block_cids_cid_fk FOREIGN KEY (block) REFERENCES public.block_cids(cid);


--
-- Name: actors id_address_map_actors_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.actors
    ADD CONSTRAINT id_address_map_actors_id_fk FOREIGN KEY (id) REFERENCES public.id_address_map(id);


--
-- Name: minerid_dealid_sectorid minerid_dealid_sectorid_sector_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.minerid_dealid_sectorid
    ADD CONSTRAINT minerid_dealid_sectorid_sector_id_fkey FOREIGN KEY (sector_id, miner_id) REFERENCES public.sector_precommit_info(sector_id, miner_id);


--
-- Name: mpool_messages mpool_messages_messages_cid_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.mpool_messages
    ADD CONSTRAINT mpool_messages_messages_cid_fk FOREIGN KEY (msg) REFERENCES public.messages(cid);


--
-- Name: minerid_dealid_sectorid sectors_sector_ids_id_fk; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.minerid_dealid_sectorid
    ADD CONSTRAINT sectors_sector_ids_id_fk FOREIGN KEY (deal_id) REFERENCES public.market_deal_proposals(deal_id);


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
-- PostgreSQL database dump complete
--
		`)

	down := batch(`
DROP MATERIALIZED VIEW IF EXISTS public.state_heights;
DROP MATERIALIZED VIEW IF EXISTS public.top_miners_by_base_reward;
DROP MATERIALIZED VIEW IF EXISTS public.top_miners_by_base_reward_max_height;
DROP FUNCTION IF EXISTS public.actor_tips;
DROP TABLE IF EXISTS public.actor_states;
DROP TABLE IF EXISTS public.actors;
DROP TABLE IF EXISTS public.block_drand_entries;
DROP TABLE IF EXISTS public.block_messages;
DROP TABLE IF EXISTS public.block_parents;
DROP TABLE IF EXISTS public.blocks;
DROP TABLE IF EXISTS public.blocks_synced;
DROP TABLE IF EXISTS public.chain_economics;
DROP TABLE IF EXISTS public.chain_power;
DROP TABLE IF EXISTS public.chain_reward;
DROP TABLE IF EXISTS public.drand_entries;
DROP TABLE IF EXISTS public.id_address_map;
DROP TABLE IF EXISTS public.market_deal_states;
DROP TABLE IF EXISTS public.miner_info;
DROP TABLE IF EXISTS public.miner_power;
DROP TABLE IF EXISTS public.miner_sector_events;
DROP TABLE IF EXISTS public.minerid_dealid_sectorid;
DROP TABLE IF EXISTS public.mpool_messages;
DROP TABLE IF EXISTS public.receipts;
DROP TABLE IF EXISTS public.sector_info;
DROP TABLE IF EXISTS public.sector_precommit_info;
DROP TABLE IF EXISTS public.block_cids;
DROP TABLE IF EXISTS public.market_deal_proposals;
DROP TABLE IF EXISTS public.messages;
DROP TYPE IF EXISTS public.miner_sector_event_type;
`)
	migrations.MustRegisterTx(up, down)
}
