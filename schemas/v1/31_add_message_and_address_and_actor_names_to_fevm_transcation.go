package v1

func init() {
	patches.Register(
		31,
		`
	ALTER TABLE {{ .SchemaName | default "public"}}.fevm_transactions
		ADD COLUMN IF NOT EXISTS "from_filecoin_address" text;
	ALTER TABLE {{ .SchemaName | default "public"}}.fevm_transactions
		ADD COLUMN IF NOT EXISTS "to_filecoin_address" text;

	ALTER TABLE {{ .SchemaName | default "public"}}.fevm_transactions
		ADD COLUMN IF NOT EXISTS "from_actor_name" text;
	ALTER TABLE {{ .SchemaName | default "public"}}.fevm_transactions
		ADD COLUMN IF NOT EXISTS "to_actor_name" text;
	
	ALTER TABLE {{ .SchemaName | default "public"}}.fevm_transactions
		ADD COLUMN IF NOT EXISTS "message_cid" text;
	
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_transactions.from_filecoin_address IS 'Filecoin Address of the sender.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_transactions.to_filecoin_address IS 'Filecoin Address of the receiver.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_transactions.from_actor_name IS 'Fully-versioned human-readable identifier of sender (From).';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_transactions.to_actor_name IS 'Fully-versioned human-readable identifier of receiver (To).';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_transactions.message_cid IS 'Filecoin Message Cid';
`,
	)
}
