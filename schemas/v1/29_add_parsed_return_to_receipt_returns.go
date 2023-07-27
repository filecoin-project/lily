package v1

func init() {
	patches.Register(
		29,
		`
	ALTER TABLE {{ .SchemaName | default "public"}}.receipts
		ADD COLUMN IF NOT EXISTS "parsed_return" JSONB;

	ALTER TABLE {{ .SchemaName | default "public"}}.receipts
		ADD COLUMN IF NOT EXISTS "return" bytea;
	
`,
	)
}
