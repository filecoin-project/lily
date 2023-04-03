package v1

func init() {
	patches.Register(
		19,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_actor_stats  (
	    height BIGINT NOT NULL,
	    contract_balance TEXT NOT NULL,
	    eth_account_balance TEXT NOT NULL,
	    placeholder_balance TEXT NOT NULL,
		contract_count BIGINT NOT NULL,
		unique_contract_count BIGINT NOT NULL,
		eth_account_count BIGINT NOT NULL,
		placeholder_count BIGINT NOT NULL,		
		PRIMARY KEY(height)
	);
`,
	)
}
