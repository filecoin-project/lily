package v1

func init() {
	patches.Register(
		27,
		`
 CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_traces (
    height BIGINT NOT NULL,
    message_state_root TEXT,
    transaction_hash TEXT,
    message_cid TEXT,
    trace_cid TEXT,
    "from" TEXT,
    "to" TEXT,
    from_filecoin_address TEXT,
    to_filecoin_address TEXT,
    value NUMERIC,
    method BIGINT,
    parsed_method TEXT,
    actor_code TEXT,
    exit_code BIGINT,
    params TEXT,
    returns TEXT,
    index BIGINT,
    parsed_params JSONB,
    parsed_returns JSONB,
    params_codec BIGINT,
    returns_codec BIGINT,
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
