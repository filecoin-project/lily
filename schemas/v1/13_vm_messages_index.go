package v1

func init() {
	patches.Register(
		13,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.vm_messages
    ADD COLUMN "index" BIGINT NOT NULL,
    ADD CONSTRAINT vm_messages_uniq_index UNIQUE (height, state_root, cid, source, index);

COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.index IS 'Order in which the message was applied.';
`,
	)
}
