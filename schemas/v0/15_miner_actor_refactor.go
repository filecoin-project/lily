package v0

// Schema version 15 adds more miner actor data

func init() {
	up := batch(`
	-- Below hypertables are height chunked per 7 days (20160 epochs)

	CREATE TABLE IF NOT EXISTS "miner_current_deadline_infos" (
		"height" bigint not null,
		"miner_id" text not null,
		"state_root" text not null,
		"deadline_index" bigint not null,
		"period_start" bigint not null,
		"open" bigint not null,
		"close" bigint not null,
		"challenge" bigint not null,
		"fault_cutoff" bigint not null,
		PRIMARY KEY ("height", "miner_id", "state_root")
	);
	SELECT create_hypertable(
		'miner_current_deadline_infos',
		'height',
		chunk_time_interval => 20160,
		if_not_exists => TRUE
	);

	CREATE TABLE IF NOT EXISTS "miner_fee_debts" (
		"height" bigint not null,
		"miner_id" text not null,
		"state_root" text not null,
		"fee_debt" text not null,
		PRIMARY KEY ("height", "miner_id", "state_root")
	);
	SELECT create_hypertable(
		'miner_fee_debts',
		'height',
		chunk_time_interval => 20160,
		if_not_exists => TRUE
	);

	CREATE TABLE IF NOT EXISTS "miner_locked_funds" (
		"height" bigint not null,
		"miner_id" text not null,
		"state_root" text not null,
		"locked_funds" text not null,
		"initial_pledge" text not null,
		"pre_commit_deposits" text not null,
		PRIMARY KEY ("height", "miner_id", "state_root")
	);
	SELECT create_hypertable(
		'miner_locked_funds',
		'height',
		chunk_time_interval => 20160,
		if_not_exists => TRUE
	);

	CREATE TABLE IF NOT EXISTS "miner_infos" (
		"height" bigint not null,
		"miner_id" text not null,
		"state_root" text not null,
		"owner_id" text not null,
		"worker_id" text not null,
		"new_worker" text,
		"worker_change_epoch" bigint not null,
		"consensus_faulted_elapsed" bigint not null,
		"peer_id" text,
		"control_addresses" jsonb,
		"multi_addresses" jsonb,
		PRIMARY KEY ("height", "miner_id", "state_root")
	);
	SELECT create_hypertable(
		'miner_infos',
		'height',
		chunk_time_interval => 20160,
		if_not_exists => TRUE
	);

	ALTER TABLE public.miner_deal_sectors RENAME TO miner_sector_deals;
`)

	down := batch(`
	DROP TABLE IF EXISTS public.miner_current_deadline_infos;
	DROP TABLE IF EXISTS public.miner_fee_debts;
	DROP TABLE IF EXISTS public.miner_locked_funds;
	DROP TABLE IF EXISTS public.miner_infos;
	ALTER TABLE public.miner_sector_deals RENAME TO miner_deal_sectors;
`)

	Patches.MustRegisterTx(up, down)
}
