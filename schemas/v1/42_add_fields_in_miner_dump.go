package v1

func init() {
	patches.Register(
		42,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.miner_actor_dumps ADD COLUMN IF NOT EXISTS termination_fee numeric DEFAULT 0;
`,
	)
}
