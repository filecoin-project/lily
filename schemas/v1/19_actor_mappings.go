package v1

// Schema version 16 adds actor_codes and actor_methods mappings

func init() {
	patches.Register(
		19,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.actor_codes (
		cid		text  NOT NULL,
		code	text  NOT NULL,
		
		PRIMARY KEY (cid, code)
	);
	COMMENT ON TABLE {{ .SchemaName | default "public"}}.actor_codes IS 'A mapping of a builtin actor's' CID to a human friendly name.';
	COMMENT ON column {{ .SchemaName | default "public"}}.actor_codes.cid IS 'CID of the actor from builtin actors.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_codes.code IS 'Human-readable identifier for the actor.';
`,
	)
}
