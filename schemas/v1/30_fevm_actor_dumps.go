package v1

func init() {
	patches.Register(
		30,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_actor_dumps  (
	    height               BIGINT NOT NULL,
		actor_id             TEXT,
		eth_address          TEXT,
		byte_code            TEXT,
		byte_code_hash       TEXT,
		balance              NUMERIC,
		nonce                BIGINT,
		actor_name           TEXT,
		PRIMARY KEY(height, actor_id, nonce)
	);
	CREATE INDEX IF NOT EXISTS fevm_actor_dumps_actor_id_idx ON {{ .SchemaName | default "public"}}.fevm_actor_dumps USING hash (actor_id);
	CREATE INDEX IF NOT EXISTS fevm_actor_dumps_eth_address_idx ON {{ .SchemaName | default "public"}}.fevm_actor_dumps USING hash (eth_address);
`,
	)
}
