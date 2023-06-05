package v1

func init() {
	patches.Register(
		26,
		`
 CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_traces (
    height BIGINT NOT NULL,
    message_state_root TEXT,
    transaction_hash TEXT,
    message_cid TEXT,
	trace_cid TEXT,
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
	PRIMARY KEY(height, message_state_root, trace_cid, message_cid)
);
CREATE INDEX IF NOT EXISTS fevm_traces_height_idx ON {{ .SchemaName | default "public"}}.fevm_traces USING BTREE (height);
CREATE INDEX IF NOT EXISTS fevm_traces_from_idx ON {{ .SchemaName | default "public"}}.fevm_traces USING HASH ("from");
CREATE INDEX IF NOT EXISTS fevm_traces_to_idx ON {{ .SchemaName | default "public"}}.fevm_traces USING HASH ("to");

SELECT create_hypertable(
	'fevm_traces',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE,
	migrate_data => TRUE
);
SELECT set_integer_now_func('fevm_traces', 'current_height', replace_if_exists => true);

`,
	)
}
