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
CREATE INDEX vm_messages_height_idx ON {{ .SchemaName | default "public"}}.vm_messages USING BTREE (height);
CREATE INDEX vm_messages_from_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH ("from");
CREATE INDEX vm_messages_to_idx ON {{ .SchemaName | default "public"}}.vm_messages USING HASH ("to");
CREATE INDEX vm_messages_actor_code_method_idx ON {{ .SchemaName | default "public"}}.vm_messages USING BTREE (actor_code, method);


COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.height IS 'Height message was executed at.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.state_root IS 'CID of the parent state root at which this message was executed.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.cid IS 'CID of the message (note this CID does not appear on chain).';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.source IS 'CID of the on-chain message that caused this message to be sent.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages."from" IS 'Address of the actor that sent the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages."to" IS 'Address of the actor that received the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.value IS 'Amount of FIL (in attoFIL) transferred by this message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.method IS 'The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method execution';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.actor_code IS 'The CID of the actor that received the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.exit_code IS 'The exit code that was returned as a result of executing the message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.gas_used IS 'A measure of the amount of resources (or units of gas) consumed, in order to execute a message.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.params IS 'Message parameters parsed and serialized as a JSON object.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.vm_messages.returns IS 'Result returned from executing a message parsed and serialized as a JSON object.';
`,
	)
}
