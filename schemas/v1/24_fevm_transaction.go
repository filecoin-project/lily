package v1

func init() {
	patches.Register(
		24,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_transaction  (
	    height               BIGINT NOT NULL,
		hash                 TEXT,
		transaction_index    BIGINT,
		block_hash           TEXT,
		block_number         BIGINT,
		chain_id             BIGINT,
		nonce                BIGINT,
		type                 BIGINT,
		"from"               TEXT,
		"to"                 TEXT,
		input                TEXT,
		value                TEXT,
		gas                  BIGINT,
		max_fee_per_gas      TEXT,
		max_priority_fee_per_gas TEXT,
		v                    TEXT,
		r                    TEXT,
		s                    TEXT,
		access_list          jsonb,
		PRIMARY KEY(height, hash)
	);
`,
	)
}
