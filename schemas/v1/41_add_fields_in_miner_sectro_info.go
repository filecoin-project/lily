package v1

func init() {
	patches.Register(
		41,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.miner_sector_infos_v7 ADD COLUMN IF NOT EXISTS replaced_day_reward numeric DEFAULT 0;
		ALTER TABLE {{ .SchemaName | default "public"}}.miner_sector_infos_v7 ADD COLUMN IF NOT EXISTS power_base_epoch bigint DEFAULT 0;
`,
	)
}
