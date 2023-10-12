package v1

func init() {
	patches.Register(
		35,
		`
		ALTER TABLE ONLY {{ .SchemaName | default "public"}}.drand_block_entries ADD CONSTRAINT drand_block_entries_pkey PRIMARY KEY (round, block);

		ALTER TABLE {{ .SchemaName | default "public"}}.vm_messages DROP CONSTRAINT IF EXISTS vm_messages_pkey CASCADE, ADD PRIMARY KEY(height, state_root, cid, source, index)
`,
	)
}
