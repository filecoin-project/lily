package v1

func init() {
	patches.Register(
		28,
		`
-- fevm_block_headers
CREATE INDEX IF NOT EXISTS fevm_block_headers_height_idx ON {{ .SchemaName | default "public"}}.fevm_block_headers USING BTREE (height);
CREATE INDEX IF NOT EXISTS fevm_block_headers_hash_idx ON {{ .SchemaName | default "public"}}.fevm_block_headers USING HASH (hash);

-- fevm_receipts
CREATE INDEX IF NOT EXISTS fevm_receipts_height_idx ON {{ .SchemaName | default "public"}}.fevm_receipts USING BTREE (height);
CREATE INDEX IF NOT EXISTS fevm_receipts_from_idx ON {{ .SchemaName | default "public"}}.fevm_receipts USING HASH ("from");
CREATE INDEX IF NOT EXISTS fevm_receipts_to_idx ON {{ .SchemaName | default "public"}}.fevm_receipts USING HASH ("to");

-- fevm_contracts
CREATE INDEX IF NOT EXISTS fevm_contracts_height_idx ON {{ .SchemaName | default "public"}}.fevm_contracts USING BTREE (height);
CREATE INDEX IF NOT EXISTS fevm_contracts_actor_id_idx ON {{ .SchemaName | default "public"}}.fevm_contracts USING HASH (actor_id);
CREATE INDEX IF NOT EXISTS fevm_contracts_eth_address_idx ON {{ .SchemaName | default "public"}}.fevm_contracts USING HASH (eth_address);

-- fevm_transactions
CREATE INDEX IF NOT EXISTS fevm_transactions_height_idx ON {{ .SchemaName | default "public"}}.fevm_transactions USING BTREE (height);
CREATE INDEX IF NOT EXISTS fevm_transactions_from_idx ON {{ .SchemaName | default "public"}}.fevm_transactions USING HASH ("from");
CREATE INDEX IF NOT EXISTS fevm_transactions_to_idx ON {{ .SchemaName | default "public"}}.fevm_transactions USING HASH ("to");
`,
	)
}
