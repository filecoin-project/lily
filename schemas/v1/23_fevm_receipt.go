package v1

func init() {
	patches.Register(
		23,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_receipt  (
	    height               BIGINT NOT NULL,
    	logs                 jsonb,
		transaction_hash     TEXT,
		transaction_index    BIGINT,
		block_hash           TEXT,
		block_number         BIGINT,
		"from"               TEXT,
		"to"                 TEXT,
		contract_address     TEXT,
		status               BIGINT,
		cumulative_gas_used  BIGINT,
		gas_used             BIGINT,
		effective_gas_price  BIGINT,
		logs_bloom           TEXT,
		message              TEXT,
		PRIMARY KEY(height, transaction_hash)
	);
`,
	)
}
