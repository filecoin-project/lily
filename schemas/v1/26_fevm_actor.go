package v1

func init() {
	patches.Register(
		26,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.fevm_actors  (
	    height               BIGINT NOT NULL,
		id                   TEXT   NOT NULL,
		eth_address          TEXT,
		state_root           TEXT,
		state                JSONB,
		code                 TEXT,
		head                 TEXT,
		code_cid             TEXT,
		PRIMARY KEY(height,id)
	);
`,
	)
}
