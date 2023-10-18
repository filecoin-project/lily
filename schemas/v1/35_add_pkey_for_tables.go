package v1

func init() {
	patches.Register(
		35,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.drand_block_entries DROP CONSTRAINT IF EXISTS drand_block_entries_pkey CASCADE, ADD PRIMARY KEY (round, block);
		ALTER TABLE {{ .SchemaName | default "public"}}.vm_messages DROP CONSTRAINT IF EXISTS vm_messages_pkey CASCADE, ADD PRIMARY KEY(height, state_root, cid, source, index);
		ALTER TABLE {{ .SchemaName | default "public"}}.actor_events ADD COLUMN IF NOT EXISTS entry_index bigint DEFAULT 0;
		ALTER TABLE {{ .SchemaName | default "public"}}.actor_events DROP CONSTRAINT IF EXISTS actor_events_pkey CASCADE, ADD PRIMARY KEY(height, state_root, message_cid, event_index, entry_index);
		ALTER TABLE {{ .SchemaName | default "public"}}.actor_states ADD COLUMN IF NOT EXISTS address text DEFAULT '';
		ALTER TABLE {{ .SchemaName | default "public"}}.actor_states DROP CONSTRAINT IF EXISTS actor_states_pkey CASCADE, ADD PRIMARY KEY(height, head, code, address);
`,
	)
}
