package v1

func init() {
	patches.Register(
		14,
		`
{{- if and .SchemaName (ne .SchemaName "public") }}
SET search_path TO {{ .SchemaName }},public;
{{- end }}

-- Convert messages to a hypertable partitioned on height (time)
-- Setting the time interval to 2880 heights so there will be one chunk per day.
SELECT create_hypertable(
	'vm_messages',
	'height',
	chunk_time_interval => 2880,
	if_not_exists => TRUE,
	migrate_data => TRUE
);
SELECT set_integer_now_func('vm_messages', 'current_height', replace_if_exists => true);
`,
	)
}
