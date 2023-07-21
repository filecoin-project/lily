package v1

func init() {
	patches.Register(
		29,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_actor_snapshots  (
	    height               BIGINT NOT NULL,
		actor_id             TEXT,
		eth_address          TEXT,
		byte_code            TEXT,
		byte_code_hash       TEXT,
		balance              NUMERIC,
		nonce                BIGINT,
		PRIMARY KEY(height, actor_id, nonce)
	);
	CREATE INDEX fevm_actor_snapshots_actor_id_idx ON {{ .SchemaName | default "public"}}.fevm_actor_snapshots USING hash (actor_id);
	CREATE INDEX fevm_actor_snapshots_eth_address_idx ON {{ .SchemaName | default "public"}}.fevm_actor_snapshots USING hash (eth_address);
`,
	)
}
