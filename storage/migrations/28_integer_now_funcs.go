package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 28 adds an integer now function to each hypertable

func init() {
	up := batch(`
	-- A function that estimates the current filecoin epoch
	CREATE OR REPLACE FUNCTION public.current_height() RETURNS bigint AS $$
		SELECT floor((extract(epoch from now() AT TIME ZONE 'UTC') - extract(epoch from TIMESTAMP '2020-8-24 22:00:00' AT TIME ZONE 'UTC')) / 30)::bigint;
	$$ LANGUAGE SQL STABLE PARALLEL SAFE;

	-- Apply the current epoch function to each hypertable to allow hypertable specific functions to be used
	SELECT set_integer_now_func('actor_states', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('actors', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('block_headers', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('block_messages', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('block_parents', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('chain_powers', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('chain_rewards', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('market_deal_proposals', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('market_deal_states', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('message_gas_economy', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('messages', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_current_deadline_infos', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_fee_debts', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_infos', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_locked_funds', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_pre_commit_infos', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_sector_deals', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_sector_events', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_sector_infos', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('miner_sector_posts', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('parsed_messages', 'current_height', replace_if_exists => true);
	SELECT set_integer_now_func('receipts', 'current_height', replace_if_exists => true);
`)

	down := batch(`

	-- set_integer_now_func is irreversible so we can't migrate down
	SELECT 1;

`)

	migrations.MustRegisterTx(up, down)
}
