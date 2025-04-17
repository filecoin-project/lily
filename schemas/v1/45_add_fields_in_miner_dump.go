package v1

func init() {
	patches.Register(
		45,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.miner_actor_dumps ADD COLUMN IF NOT EXISTS termination_fee_v2 numeric DEFAULT 0;
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_actor_dumps IS 'A penalty imposed when a sector is prematurely terminated in attoFIL (after nv25).';
`,
	)
}
