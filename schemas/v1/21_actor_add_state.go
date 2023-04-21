package v1

func init() {
	patches.Register(
		21,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.actors
    ADD COLUMN IF NOT EXISTS "state" jsonb;

ALTER TABLE {{ .SchemaName | default "public"}}.actors
	ADD COLUMN IF NOT EXISTS "code_cid" text;

COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.state IS 'Top level of state data.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actors.code_cid IS 'CID identifier for the type of the actor.';
`,
	)
}
