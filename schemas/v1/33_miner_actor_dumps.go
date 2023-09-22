package v1

func init() {
	patches.Register(
		33,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.miner_actor_dumps  (
	    height               BIGINT NOT NULL,
		miner_id             TEXT,
		miner_address        TEXT,
		state_root           TEXT,
		owner_id             TEXT,
		worker_id            TEXT,

		consensus_faulted_elapsed BIGINT,

		peer_id              TEXT,
		control_addresses    JSONB,
		beneficiary          TEXT,

		sector_size          BIGINT,
		num_live_sectors     BIGINT,

		raw_byte_power       NUMERIC,
		quality_adj_power    NUMERIC,
		total_locked_funds   NUMERIC,
		vesting_funds        NUMERIC,
		initial_pledge       NUMERIC,
		pre_commit_deposits  NUMERIC,
		available_balance    NUMERIC,
		balance              NUMERIC,
		fee_debt             NUMERIC,
		
		PRIMARY KEY(height, miner_id, miner_address)
	);
	CREATE INDEX IF NOT EXISTS miner_actor_dumps_height_idx ON {{ .SchemaName | default "public"}}.miner_actor_dumps USING btree (height);
	CREATE INDEX IF NOT EXISTS miner_actor_dumps_miner_id_idx ON {{ .SchemaName | default "public"}}.miner_actor_dumps USING hash (miner_id);
	CREATE INDEX IF NOT EXISTS miner_actor_dumps_miner_address_idx ON {{ .SchemaName | default "public"}}.miner_actor_dumps USING hash (miner_address);
`,
	)
}
