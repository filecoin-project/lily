package v1

func init() {
	patches.Register(
		16,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.receipts
	ADD COLUMN events_root TEXT;

COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipts.events_root IS 'The root of AMT<StampedEvent, bitwidth=5>. It is null when no events have been emitted by an actor. See FIP-0049.';
`)
}
