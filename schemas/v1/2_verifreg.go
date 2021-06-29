package v1

// Schema version 1 adds verified registry actor state tracking

func init() {
	patches.Register(
		2,
		`
	CREATE TYPE {{ .SchemaName | default "public"}}.verified_registry_event_type AS ENUM (
		'ADDED',
		'REMOVED',
		'MODIFIED'
	);
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.verified_registry_verifiers (
		"height"		bigint  NOT NULL,
		"state_root"	text    NOT NULL,
		"address"		text 	NOT NULL,

		"data_cap" 		numeric NOT NULL,
		"event" 		{{ .SchemaName | default "public"}}.verified_registry_event_type NOT NULL,

		PRIMARY KEY ("height", "state_root", "address")
	);
	COMMENT ON TABLE {{ .SchemaName | default "public"}}.verified_registry_verifiers IS 'Verifier on-chain per each verifier state change.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verifiers.height IS 'Epoch at which this verifiers state changed.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verifiers.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verifiers.address IS 'Address of verifier this state change applies to.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verifiers.data_cap IS 'DataCap of verifier at this state change.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verifiers.event IS 'Name of the event that occurred.';

	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.verified_registry_verified_clients (
		"height"		bigint  NOT NULL,
		"state_root"	text    NOT NULL,
		"address"		text 	NOT NULL,

		"data_cap" 		numeric NOT NULL,
		"event" 		{{ .SchemaName | default "public"}}.verified_registry_event_type NOT NULL,

		PRIMARY KEY ("height", "state_root", "address")
	);
	COMMENT ON TABLE {{ .SchemaName | default "public"}}.verified_registry_verified_clients IS 'Verifier on-chain per each verified client state change.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verified_clients.height IS 'Epoch at which this verified client state changed.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verified_clients.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verified_clients.address IS 'Address of verified client this state change applies to.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verified_clients.data_cap IS 'DataCap of verified client at this state change.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.verified_registry_verified_clients.event IS 'Name of the event that occurred.';
`)
}
