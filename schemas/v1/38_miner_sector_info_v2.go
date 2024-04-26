package v1

func init() {
	patches.Register(
		38,
		`
	CREATE TABLE {{ .SchemaName | default "public"}}.miner_sector_deals_v2 (
		miner_id text NOT NULL,
		sector_id bigint NOT NULL,
		deal_id bigint NOT NULL,
		height bigint NOT NULL
	);
	ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_sector_deals_v2 ADD CONSTRAINT miner_sector_deals_v2_pkey PRIMARY KEY (height, miner_id, sector_id, deal_id);
	CREATE INDEX IF NOT EXISTS miner_sector_deals_height_idx ON {{ .SchemaName | default "public"}}.miner_sector_deals_v2 USING btree (height DESC);
	CREATE INDEX IF NOT EXISTS miner_sector_deals_miner_id_idx ON {{ .SchemaName | default "public"}}.miner_sector_deals_v2 USING hash (miner_id);
	CREATE INDEX IF NOT EXISTS miner_sector_deals_sector_id_idx ON {{ .SchemaName | default "public"}}.miner_sector_deals_v2 USING hash (sector_id);
`,
	)
}
