package v1

func init() {
	patches.Register(
		26,
		`
 CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_vm_messages (
    height BIGINT NOT NULL,
    state_root TEXT,
    transaction_hash TEXT,
    cid TEXT,
	source TEXT,
    "from" TEXT,
    "to" TEXT,
    from_eth_address TEXT,
    to_eth_address TEXT,
    value NUMERIC,
    method BIGINT,
    actor_code TEXT,
    exit_code BIGINT,
    gas_used BIGINT,
    params TEXT,
	returns TEXT,
	index BIGINT,
    parsed_params JSONB,
	parsed_returns JSONB,
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
