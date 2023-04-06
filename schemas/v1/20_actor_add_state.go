package v1

func init() {
	patches.Register(
		20,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.actors
    ADD COLUMN IF NOT EXISTS "state" jsonb;

COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.state IS 'Top level of state data.';
`,
	)
}
