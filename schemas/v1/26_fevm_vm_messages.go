package v1

func init() {
	patches.Register(
		26,
		`
 CREATE TABLE {{ .SchemaName | default "public"}}.fevm_vm_messages (
    height bigint NOT NULL,
    state_root text,
    transaction_hash text,
    cid text,
	source text,
    "from" text,
    "to" text,
    from_eth_address text,
    to_eth_address text,
    value numeric,
    method bigint,
    actor_code text,
    exit_code bigint,
    gas_used bigint,
    params text,
	returns text,
	index bigint,
    parsed_params jsonb,
	parsed_returns jsonb,
	PRIMARY KEY(height, state_root, cid, source)
);
CREATE INDEX IF NOT EXISTS fevm_vm_messages_height_idx ON {{ .SchemaName | default "public"}}.fevm_vm_messages USING BTREE (height);
CREATE INDEX IF NOT EXISTS fevm_vm_messages_from_idx ON {{ .SchemaName | default "public"}}.fevm_vm_messages USING HASH ("from");
CREATE INDEX IF NOT EXISTS fevm_vm_messages_to_idx ON {{ .SchemaName | default "public"}}.fevm_vm_messages USING HASH ("to");

SELECT create_hypertable(
	'fevm_vm_messages',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE,
	migrate_data => TRUE
);
SELECT set_integer_now_func('fevm_vm_messages', 'current_height', replace_if_exists => true);

`,
	)
}
