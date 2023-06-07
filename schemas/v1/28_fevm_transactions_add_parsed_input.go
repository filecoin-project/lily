package v1

func init() {
	patches.Register(
		28,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.fevm_transactions
    ADD COLUMN IF NOT EXISTS "parsed_input" jsonb;
`,
	)
}
