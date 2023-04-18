package v1

func init() {
	patches.Register(
		20,
		`
CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.actor_codes (
	cid		text  NOT NULL,
	code	text  NOT NULL,
	
	PRIMARY KEY (cid, code)
);
COMMENT ON TABLE {{ .SchemaName | default "public"}}.actor_codes IS 'A mapping of a builtin actors CID to a human friendly name.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_codes.cid IS 'CID of the actor from builtin actors.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_codes.code IS 'Human-readable identifier for the actor.';
    
CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.actor_methods (
	family		text  NOT NULL,
	method_name	text  NOT NULL,
	method  	bigint NOT NULL,
	
	PRIMARY KEY (family, method)
);
COMMENT ON TABLE {{ .SchemaName | default "public"}}.actor_methods IS 'A mapping of a builtin actors CID to a human friendly name.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_methods.family IS 'The actor family.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_methods.method_name IS 'Human-readable identifier for the actor method.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_methods.method IS 'Method as bigint.';
`,
	)
}
