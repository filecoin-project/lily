package v1

func init() {
	patches.Register(
		33,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.unsynced_block_headers (
		height            BIGINT NOT NULL,
		cid               TEXT NOT NULL,
		miner             TEXT,
		parent_weight     TEXT,
		parent_base_fee   TEXT,
		parent_state_root TEXT,
		win_count         BIGINT,
		"timestamp"       BIGINT,
		fork_signaling    BIGINT,
		PRIMARY KEY(height, cid)
	);
	CREATE INDEX IF NOT EXISTS unsynced_block_headers_height_idx ON {{ .SchemaName | default "public"}}.unsynced_block_headers USING btree (height DESC);
	CREATE INDEX IF NOT EXISTS unsynced_block_headers_timestamp_idx ON {{ .SchemaName | default "public"}}.unsynced_block_headers USING btree ("timestamp");
	CREATE INDEX IF NOT EXISTS unsynced_block_headers_miner_idx ON {{ .SchemaName | default "public"}}.unsynced_block_headers USING hash (miner);
`,
	)
}
