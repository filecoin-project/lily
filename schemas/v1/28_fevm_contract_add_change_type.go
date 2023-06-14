package v1

func init() {
	patches.Register(
		28,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.fevm_contracts
    ADD COLUMN IF NOT EXISTS "change_type" TEXT;

COMMENT ON COLUMN {{ .SchemaName | default "public"}}.fevm_contracts.change_type IS 'Contract change type: Add, Remove, Modify and Unknown';
`,
	)
}
