package v1

func init() {
	patches.Register(
		45,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.miner_actor_dumps ADD COLUMN IF NOT EXISTS termination_fee_v2 numeric DEFAULT 0;
`,
	)
}
