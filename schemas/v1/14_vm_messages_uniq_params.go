package v1

func init() {
	patches.Register(
		14,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.vm_messages
  ADD CONSTRAINT vm_messages_uniq_params UNIQUE (height, state_root, cid, source, params);
`,
	)
}
