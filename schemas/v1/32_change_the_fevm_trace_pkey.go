package v1

func init() {
	patches.Register(
		32,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.fevm_traces DROP CONSTRAINT IF EXISTS fevm_traces_pkey CASCADE, ADD PRIMARY KEY(height, index, message_cid)
`,
	)
}
