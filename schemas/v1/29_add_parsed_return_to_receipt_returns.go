package v1

func init() {
	patches.Register(
		29,
		`
	ALTER TABLE {{ .SchemaName | default "public"}}.receipt_returns
		ADD COLUMN IF NOT EXISTS "parsed_return" JSONB;
	
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_traces.from_actor_name IS 'Fully-versioned human-readable identifier of receiver (From).';
`,
	)
}
