package v1

func init() {
	patches.Register(
		37,
		`
	ALTER TABLE {{ .SchemaName | default "public"}}.builtin_actor_events ADD COLUMN IF NOT EXISTS "event_idx" BIGINT DEFAULT 0;
	ALTER TABLE {{ .SchemaName | default "public"}}.builtin_actor_events DROP CONSTRAINT IF EXISTS builtin_actor_events_pkey CASCADE, ADD PRIMARY KEY(height, cid, emitter, event_type, event_idx);

	CREATE INDEX IF NOT EXISTS builtin_actor_events_emitter_idx ON {{ .SchemaName | default "public"}}.builtin_actor_events USING hash (emitter);
	CREATE INDEX IF NOT EXISTS builtin_actor_events_event_type_idx ON {{ .SchemaName | default "public"}}.builtin_actor_events USING hash (event_type);
	CREATE INDEX IF NOT EXISTS builtin_actor_events_height_idx ON {{ .SchemaName | default "public"}}.builtin_actor_events USING btree (height);

	SELECT create_hypertable(
		'builtin_actor_events',
		'height',
		chunk_time_interval => 2880,
		if_not_exists => TRUE,
		migrate_data => TRUE
	);
	SELECT set_integer_now_func('builtin_actor_events', 'current_height', replace_if_exists => true);
`,
	)
}
