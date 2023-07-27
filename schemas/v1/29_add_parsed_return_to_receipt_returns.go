package v1

func init() {
	patches.Register(
		29,
		`
	ALTER TABLE {{ .SchemaName | default "public"}}.receipt_returns
		ADD COLUMN IF NOT EXISTS "parsed_return" JSONB;
	
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipt_returns.parsed_return IS 'Result returned from executing a message parsed and serialized as a JSON object.';
`,
	)
}
