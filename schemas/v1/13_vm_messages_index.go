package v1

func init() {
	patches.Register(
		13,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.vm_messages
    ADD COLUMN IF NOT EXISTS "index" BIGINT NOT NULL DEFAULT -1;

ALTER TABLE {{ .SchemaName | default "public"}}.vm_messages ALTER COLUMN "index" DROP DEFAULT;

COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.index IS 'Order in which the message was applied.';
`,
	)
}
