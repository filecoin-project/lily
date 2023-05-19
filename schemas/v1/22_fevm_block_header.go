package v1

func init() {
	patches.Register(
		22,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_block_header  (
	    height BIGINT NOT NULL,
		hash TEXT,
		parent_hash TEXT,
		miner TEXT,
		state_root TEXT,
		transactions_root TEXT,
		receipts_root TEXT,
		difficulty BIGINT,
		number BIGINT,
		gas_limit BIGINT,
		gas_used BIGINT,
		timestamp BIGINT,
		extra_data TEXT,
		mix_hash TEXT,
		nonce TEXT,
		base_fee_per_gas TEXT,
		size BIGINT,
		sha3_uncles TEXT,
		PRIMARY KEY(height)
	);
`,
	)
}
