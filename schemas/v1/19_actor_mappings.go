package v1

// Schema version 16 adds actor_codes and actor_methods mappings

func init() {
	patches.Register(
		19,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.actor_codes (
		cid		text  NOT NULL,
		code	text  NOT NULL,
		
		PRIMARY KEY (cid)
	);
	COMMENT ON TABLE {{ .SchemaName | default "public"}}.actor_codes IS 'A mapping of builtin actors CIDs and human friendly names';
	COMMENT ON column {{ .SchemaName | default "public"}}.actor_codes.cid IS 'CID of the actor from builtin actors.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_codes.code IS 'Human-readable identifier for the actor.';
`,
	)
}
