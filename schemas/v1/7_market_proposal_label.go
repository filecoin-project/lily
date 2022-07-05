package v1

func init() {
	patches.Register(
		7,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.market_deal_proposals
ADD COLUMN is_string BOOLEAN;

COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals.is_string IS 'When true Label contains a valid UTF-8 string encoded in base64. When false Label contains raw bytes encoded in base64. Required by FIP: 27';
`,
	)
}
