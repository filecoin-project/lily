package v1

// Schema version 11 adds DataCap actor balance tracking

func init() {
	patches.Register(
		12,
		`
	CREATE TYPE {{ .SchemaName | default "public"}}.data_cap_balance_event_type AS ENUM (
		'ADDED',
		'REMOVED',
		'MODIFIED'
	);
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.data_cap_balances (
		"height"		bigint  NOT NULL,
		"state_root"	text    NOT NULL,
		"address"		text 	NOT NULL,

		"data_cap" 		numeric NOT NULL,
		"event" 		{{ .SchemaName | default "public"}}.data_cap_balance_event_type NOT NULL,

		PRIMARY KEY ("height", "state_root", "address")
	);
	COMMENT ON TABLE {{ .SchemaName | default "public"}}.data_cap_balances IS 'DataCap balances on-chain per each DataCap state change.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.data_cap_balances.height IS 'Epoch at which DataCap balances state changed.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.data_cap_balances.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.data_cap_balances.address IS 'Address of verified datacap client this state change applies to.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.data_cap_balances.data_cap IS 'DataCap of verified datacap client at this state change.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.data_cap_balances.event IS 'Name of the event that occurred (ADDED, MODIFIED, REMOVED).';
`,
	)
}
