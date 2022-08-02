package v1

func init() {
	patches.Register(
		8,
		`
 CREATE TABLE {{ .SchemaName | default "public"}}.vm_messages (
    height bigint NOT NULL,
    state_root text NOT NULL,
    cid text NOT NULL,
	source text,
    "from" text NOT NULL,
    "to" text NOT NULL,
    value numeric NOT NULL,
    method bigint NOT NULL,
    actor_code text NOT NULL,
    exit_code bigint NOT NULL,
    gas_used bigint NOT NULL,
    params jsonb,
	returns jsonb
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.vm_messages ADD CONSTRAINT vm_messages_pkey PRIMARY KEY (height, state_root, cid, source);
CREATE INDEX vm_messages_height_idx ON {{ .SchemaName | default "public"}}.vm_messages USING btree (height DESC);
CREATE INDEX vm_messages_state_root_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH (state_root);
CREATE INDEX vm_messages_cid_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH (cid);
CREATE INDEX vm_messages_source_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH (source);
CREATE INDEX vm_messages_from_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH ("from");
CREATE INDEX vm_messages_to_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH ("to");
CREATE INDEX vm_messages_method_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH (method);
CREATE INDEX vm_messages_actor_code_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH (actor_code);
`,
	)
}
