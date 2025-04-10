package v1

func init() {
	patches.Register(
		44,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.miner_actor_dumps ADD COLUMN IF NOT EXISTS daily_fee numeric DEFAULT 0;
`,
	)
}
