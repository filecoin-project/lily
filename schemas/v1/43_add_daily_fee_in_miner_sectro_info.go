package v1

func init() {
	patches.Register(
		43,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.miner_sector_infos_v7 ADD COLUMN IF NOT EXISTS daily_fee numeric DEFAULT 0;
`,
	)
}
