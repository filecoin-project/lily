package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 19 adds the visorstats scheme with views that show the progress of visor over ranges of epochs

func init() {
	up := batch(`

-- A function that estimates the current filecoin epoch
CREATE OR REPLACE FUNCTION public.current_epoch() RETURNS bigint
AS
$body$
SELECT floor((extract(epoch from now() AT TIME ZONE 'UTC') - extract(epoch from TIMESTAMP '2020-8-24 22:00:00' AT TIME ZONE 'UTC')) / 30)::bigint;
$body$
language sql STABLE;

-- Apply the current epoch function to each hypertable to allow continuous aggregates to be created

SELECT set_integer_now_func('visor_processing_tipsets', 'current_epoch');
SELECT set_integer_now_func('visor_processing_messages', 'current_epoch');
SELECT set_integer_now_func('visor_processing_actors', 'current_epoch');
SELECT set_integer_now_func('actor_states', 'current_epoch');
SELECT set_integer_now_func('actors', 'current_epoch');
SELECT set_integer_now_func('block_headers', 'current_epoch');
SELECT set_integer_now_func('block_messages', 'current_epoch');
SELECT set_integer_now_func('block_parents', 'current_epoch');
SELECT set_integer_now_func('chain_powers', 'current_epoch');
SELECT set_integer_now_func('chain_rewards', 'current_epoch');
SELECT set_integer_now_func('market_deal_proposals', 'current_epoch');
SELECT set_integer_now_func('market_deal_states', 'current_epoch');
SELECT set_integer_now_func('message_gas_economy', 'current_epoch');
SELECT set_integer_now_func('messages', 'current_epoch');
SELECT set_integer_now_func('miner_current_deadline_infos', 'current_epoch');
SELECT set_integer_now_func('miner_fee_debts', 'current_epoch');
SELECT set_integer_now_func('miner_infos', 'current_epoch');
SELECT set_integer_now_func('miner_locked_funds', 'current_epoch');
SELECT set_integer_now_func('miner_powers', 'current_epoch');
SELECT set_integer_now_func('miner_pre_commit_infos', 'current_epoch');
SELECT set_integer_now_func('miner_sector_deals', 'current_epoch');
SELECT set_integer_now_func('miner_sector_events', 'current_epoch');
SELECT set_integer_now_func('miner_sector_infos', 'current_epoch');
SELECT set_integer_now_func('miner_sector_posts', 'current_epoch');
SELECT set_integer_now_func('miner_states', 'current_epoch');
SELECT set_integer_now_func('parsed_messages', 'current_epoch');
SELECT set_integer_now_func('receipts', 'current_epoch');


CREATE SCHEMA IF NOT EXISTS visorstats;

DROP VIEW IF EXISTS visorstats.visor_processing_tipsets CASCADE;
CREATE VIEW visorstats.visor_processing_tipsets( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.visor_processing_tipsets GROUP BY lower;

DROP VIEW IF EXISTS visorstats.visor_processing_messages CASCADE;
CREATE VIEW visorstats.visor_processing_messages( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.visor_processing_messages GROUP BY lower;

DROP VIEW IF EXISTS visorstats.visor_processing_actors CASCADE;
CREATE VIEW visorstats.visor_processing_actors( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.visor_processing_actors GROUP BY lower;

DROP VIEW IF EXISTS visorstats.actor_states CASCADE;
CREATE VIEW visorstats.actor_states( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.actor_states GROUP BY lower;

DROP VIEW IF EXISTS visorstats.actors CASCADE;
CREATE VIEW visorstats.actors( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.actors GROUP BY lower;

DROP VIEW IF EXISTS visorstats.block_headers CASCADE;
CREATE VIEW visorstats.block_headers( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.block_headers GROUP BY lower;

DROP VIEW IF EXISTS visorstats.block_messages CASCADE;
CREATE VIEW visorstats.block_messages( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.block_messages GROUP BY lower;

DROP VIEW IF EXISTS visorstats.block_parents CASCADE;
CREATE VIEW visorstats.block_parents( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.block_parents GROUP BY lower;

DROP VIEW IF EXISTS visorstats.chain_powers CASCADE;
CREATE VIEW visorstats.chain_powers( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.chain_powers GROUP BY lower;

DROP VIEW IF EXISTS visorstats.chain_rewards CASCADE;
CREATE OR REPLACE VIEW visorstats.chain_rewards( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.chain_rewards GROUP BY lower;

DROP VIEW IF EXISTS visorstats.market_deal_proposals CASCADE;
CREATE VIEW visorstats.market_deal_proposals( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.market_deal_proposals GROUP BY lower;

DROP VIEW IF EXISTS visorstats.market_deal_states CASCADE;
CREATE VIEW visorstats.market_deal_states( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.market_deal_states GROUP BY lower;

DROP VIEW IF EXISTS visorstats.message_gas_economy CASCADE;
CREATE VIEW visorstats.message_gas_economy( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.message_gas_economy GROUP BY lower;

DROP VIEW IF EXISTS visorstats.messages CASCADE;
CREATE VIEW visorstats.messages( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.messages GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_current_deadline_infos CASCADE;
CREATE VIEW visorstats.miner_current_deadline_infos( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_current_deadline_infos GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_fee_debts CASCADE;
CREATE VIEW visorstats.miner_fee_debts( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_fee_debts GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_infos CASCADE;
CREATE VIEW visorstats.miner_infos( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_infos GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_locked_funds CASCADE;
CREATE VIEW visorstats.miner_locked_funds( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_locked_funds GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_powers CASCADE;
CREATE VIEW visorstats.miner_powers( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_powers GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_pre_commit_infos CASCADE;
CREATE VIEW visorstats.miner_pre_commit_infos( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_pre_commit_infos GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_sector_deals CASCADE;
CREATE VIEW visorstats.miner_sector_deals( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_sector_deals GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_sector_events CASCADE;
CREATE VIEW visorstats.miner_sector_events( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_sector_events GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_sector_infos CASCADE;
CREATE VIEW visorstats.miner_sector_infos( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_sector_infos GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_sector_posts CASCADE;
CREATE VIEW visorstats.miner_sector_posts( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_sector_posts GROUP BY lower;

DROP VIEW IF EXISTS visorstats.miner_states CASCADE;
CREATE VIEW visorstats.miner_states( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.miner_states GROUP BY lower;

DROP VIEW IF EXISTS visorstats.parsed_messages CASCADE;
CREATE VIEW visorstats.parsed_messages( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.parsed_messages GROUP BY lower;

DROP VIEW IF EXISTS visorstats.receipts CASCADE;
CREATE VIEW visorstats.receipts( lower, upper, count )
WITH ( timescaledb.continuous, timescaledb.refresh_lag = 10, timescaledb.refresh_interval = '15m' )
AS
SELECT time_bucket(bigint '10000', height) AS lower, max(height) as upper, count(*) FROM public.receipts GROUP BY lower;

`)

	down := batch(`SELECT 1;`)

	migrations.MustRegisterTx(up, down)
}
