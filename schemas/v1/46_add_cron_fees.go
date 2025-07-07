package v1

func init() {
	patches.Register(
		46,
		`
		CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.miner_cron_fees (
			height bigint NOT NULL,
			address text NOT NULL,
			burn numeric DEFAULT 0,
			fee numeric DEFAULT 0,
			penalty numeric DEFAULT 0
		);
		ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_cron_fees ADD CONSTRAINT miner_cron_fees_pk PRIMARY KEY (height, address);

		CREATE INDEX IF NOT EXISTS miner_cron_fees_height_idx ON {{ .SchemaName | default "public"}}.miner_cron_fees USING btree (height DESC);
		`,
	)
}
