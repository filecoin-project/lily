package v1

func init() {
	patches.Register(
		26,
		`
ALTER TABLE IF EXISTS {{ .SchemaName | default "public"}}.fevm_block_header RENAME TO fevm_block_headers;
ALTER TABLE IF EXISTS {{ .SchemaName | default "public"}}.fevm_receipt RENAME TO fevm_receipts;
ALTER TABLE IF EXISTS {{ .SchemaName | default "public"}}.fevm_transaction RENAME TO fevm_transactions;
`,
	)
}
