package v1

func init() {
	patches.Register(
		36,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.builtin_actor_events  (
	    height               BIGINT NOT NULL,
		cid                  TEXT,
		emitter          	 TEXT,
		event_type           TEXT,
		event_entries        JSONB,
		event_payload        JSONB,
		PRIMARY KEY(height, cid, emitter, event_type)
	);
`,
	)
}
