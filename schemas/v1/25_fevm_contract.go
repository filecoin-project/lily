package v1

func init() {
	patches.Register(
		25,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_contract  (
	    height               BIGINT NOT NULL,
		actor_id             TEXT,
		eth_address          TEXT,
		byte_code            TEXT,
		byte_code_hash       TEXT,
		balance              numeric,
		nonce                BIGINT,
		PRIMARY KEY(height, actor_id)
	);
`,
	)
}
