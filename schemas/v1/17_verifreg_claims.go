package v1

func init() {
	patches.Register(
		17,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.verified_registry_claims  (
	    height BIGINT NOT NULL,
	    state_root TEXT NOT NULL,
		provider TEXT NOT NULL,
		claim_id BIGINT NOT NULL,
		client TEXT NOT NULL,
		data TEXT NOT NULL,
		size BIGINT NOT NULL,
		term_min BIGINT NOT NULL,
		term_max BIGINT NOT NULL,
		term_start BIGINT NOT NULL,
		sector BIGINT NOT NULL,
		event 		{{ .SchemaName | default "public"}}.verified_registry_event_type NOT NULL,
		
		PRIMARY KEY(height, state_root, provider, claim_id)
	);
`,
	)
}
