package v1

func init() {
	patches.Register(
		14,
		`
{{- if and .SchemaName (ne .SchemaName "public") }}
SET search_path TO {{ .SchemaName }},public;
{{- end }}

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
