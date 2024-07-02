package v1

func init() {
	patches.Register(
		39,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.chain_economics ADD COLUMN IF NOT EXISTS locked_fil_v2 numeric DEFAULT 0;
`,
	)
}
