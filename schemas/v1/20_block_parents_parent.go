package v1

func init() {
	patches.Register(
		20,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.block_parents
	ALTER COLUMN "parent"  SET DEFAULT '';
`,
	)
}
