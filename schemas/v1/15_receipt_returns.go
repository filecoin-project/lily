package v1

// Schema version 14 adds receipt return tracking

func init() {
	patches.Register(
		15,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.receipt_returns (
		message		text  NOT NULL,
		"return"	bytea,
		
		PRIMARY KEY (message)
	);
	COMMENT ON TABLE {{ .SchemaName | default "public"}}.receipt_returns IS 'Raw parameters of on chain receipt.';
	COMMENT ON column {{ .SchemaName | default "public"}}.receipt_returns.message IS 'The CID of the message that produced in this receipt.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.receipt_returns.return IS 'The return of the receipt as bytes.';
`,
	)
}
