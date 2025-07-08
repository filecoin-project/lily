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

		COMMENT ON TABLE {{ .SchemaName | default "public"}}.miner_cron_fees IS 'Miner cron fees table, storing fees and penalties for miner cron events.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_cron_fees.height IS 'Height of the tipset where the cron event occurred.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_cron_fees.address IS 'Address of the miner that was involved in the cron event.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_cron_fees.burn IS 'Amount of FIL burned during the cron event.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_cron_fees.fee IS 'Amount of FIL charged as a fee during the cron event.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.miner_cron_fees.penalty IS 'Amount of FIL penalized during the cron event.';
		`,
	)
}
