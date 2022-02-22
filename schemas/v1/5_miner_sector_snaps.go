package v1

func init() {
	patches.Register(
		5,
		`
	-- add new sector event type for snapped sectors
	ALTER TYPE {{ .SchemaName | default "public"}}.miner_sector_event_type ADD VALUE 'SECTOR_SNAPPED' AFTER 'SECTOR_TERMINATED';

	-- ----------------------------------------------------------------
	-- Name: miner_sector_infos_v7
	-- Model: miner.MinerSectorInfoV7Plus
	-- Growth: About 180 per epoch
	-- ----------------------------------------------------------------
	CREATE TABLE {{ .SchemaName | default "public"}}.miner_sector_infos_v7 (
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
		height bigint NOT NULL,
		sector_key_cid text
	);
	ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_sector_infos_v7 ADD CONSTRAINT miner_sector_infos_v7_pkey PRIMARY KEY (height, miner_id, sector_id, state_root);
	CREATE INDEX miner_sector_infos_v7_height_idx ON {{ .SchemaName | default "public"}}.miner_sector_infos_v7 USING btree (height DESC);

	-- Convert miner_sector_infos_v7 to a hypertable partitioned on height (time)
	-- Assume ~180 per epoch, ~300 bytes per table row
	-- Height chunked per 7 days so we expect 20160*5 = ~3628800 rows per chunk, ~1GiB per chunk
	SELECT create_hypertable(
		'miner_sector_infos_v7',
		'height',
		chunk_time_interval => 20160,
		if_not_exists => TRUE
	);
	SELECT set_integer_now_func('miner_sector_infos_v7', 'current_height', replace_if_exists => true);

	COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_sector_infos_v7 IS 'Latest state of sectors by Miner for actors v7 and above.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.miner_id IS 'Address of the miner who owns the sector.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.sector_id IS 'Numeric identifier of the sector.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.sealed_cid IS 'The root CID of the Sealed Sector’s merkle tree. Also called CommR, or "replica commitment".';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.activation_epoch IS 'Epoch during which the sector proof was accepted.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.expiration_epoch IS 'Epoch during which the sector expires.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.deal_weight IS 'Integral of active deals over sector lifetime.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.verified_deal_weight IS 'Integral of active verified deals over sector lifetime.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.initial_pledge IS 'Pledge collected to commit this sector (in attoFIL).';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.expected_day_reward IS 'Expected one day projection of reward for sector computed at activation time (in attoFIL).';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.expected_storage_pledge IS 'Expected twenty day projection of reward for sector computed at activation time (in attoFIL).';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.height IS 'Epoch at which this sector info was added/updated.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_sector_infos_v7.sector_key_cid IS 'SealedSectorCID is set when CC sector is snapped.';
`)
}
