package v1

// Schema version 13 adds message params tracking

func init() {
	patches.Register(
		14,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.message_params (
		cid		text  NOT NULL,
    	params 	bytea,

		PRIMARY KEY (cid)
	);
	COMMENT ON TABLE {{ .SchemaName | default "public"}}.message_params IS 'Raw parameters of on chain messages.';
	COMMENT ON column {{ .SchemaName | default "public"}}.message_params.cid IS 'The CID of a message.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.message_params.params IS 'The parameters of the message as bytes.';
`,
	)
}
