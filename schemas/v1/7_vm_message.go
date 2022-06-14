package v1

func init() {
	patches.Register(
		7,
		`
CREATE TABLE {{ .SchemaName | default "public"}}.vm_messages (
    height bigint NOT NULL,
    state_root text NOT NULL,
    cid text NOT NULL,
	parent text,
    "from" text NOT NULL,
    "to" text NOT NULL,
    value numeric NOT NULL,
    method text NOT NULL,
    actor_name text NOT NULL,
    exit_code bigint NOT NULL,
    gas_used bigint NOT NULL,
    params jsonb,
	return jsonb
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.vm_messages ADD CONSTRAINT vm_messages_pkey PRIMARY KEY (height, state_root, cid);
CREATE INDEX vm_messages_exit_code_index ON {{ .SchemaName | default "public"}}.vm_messages USING btree (exit_code);
CREATE INDEX vm_messages_from_index ON {{ .SchemaName | default "public"}}.vm_messages USING hash ("from");
CREATE INDEX vm_messages_method_index ON {{ .SchemaName | default "public"}}.vm_messages USING btree (method);
CREATE INDEX vm_messages_to_index ON {{ .SchemaName | default "public"}}.vm_messages USING hash ("to");
CREATE INDEX vm_messages_actor_name_index ON {{ .SchemaName | default "public"}}.vm_messages USING btree ("actor_name");
`)
}
