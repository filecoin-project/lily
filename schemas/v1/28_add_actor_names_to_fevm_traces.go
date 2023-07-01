package v1

func init() {
	patches.Register(
		28,
		`
	ALTER TABLE {{ .SchemaName | default "public"}}.fevm_traces
		ADD COLUMN IF NOT EXISTS "to_actor_code" text NOT NULL;
	
	ALTER TABLE {{ .SchemaName | default "public"}}.fevm_traces
		ADD COLUMN IF NOT EXISTS "from_actor_code" text NOT NULL;
	
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_traces.to_actor_code IS 'Fully-versioned human-readable identifier of receiver (To).';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_traces.from_actor_code IS 'Fully-versioned human-readable identifier of receiver (From).';
`,
	)
}
