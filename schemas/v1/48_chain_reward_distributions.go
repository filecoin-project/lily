package v1

func init() {
	patches.Register(
		48,
		`
		CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.chain_reward_distributions (
			height bigint NOT NULL,
			tipset_key text NOT NULL,
			distribution jsonb NOT NULL
		);
		ALTER TABLE ONLY {{ .SchemaName | default "public"}}.chain_reward_distributions ADD CONSTRAINT chain_reward_distributions_pk PRIMARY KEY (height, tipset_key);

		CREATE INDEX IF NOT EXISTS chain_reward_distributions_height_idx ON {{ .SchemaName | default "public"}}.chain_reward_distributions USING btree (height DESC);

		COMMENT ON TABLE {{ .SchemaName | default "public"}}.chain_reward_distributions IS 'FIP-0118 block reward distribution produced by executing a tipset, as returned by Lotus StateRewardDistribution. Amounts are attoFIL decimal strings; weights and shares use the denom in the JSON.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_distributions.height IS 'Epoch of the executed tipset.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_distributions.tipset_key IS 'Key of the executed tipset.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_distributions.distribution IS 'Reward distribution as json: Totals plus per-block, per-stream and per-recipient amounts, weights and shares.';
		`,
	)
}
