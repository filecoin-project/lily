package v1

// Schema patch 11 adds miner beneficiary

func init() {
	patches.Register(
		11,
		`
CREATE TABLE {{ .SchemaName | default "public"}}.miner_beneficiaries (
    height bigint NOT NULL,
    state_root text NOT NULL,
    miner_id text NOT NULL,

	beneficiary text NOT NULL,

	quota numeric NOT NULL,
	used_quota numeric NOT NULL,
	expiration bigint NOT NULL,

	new_beneficiary text,
	new_quota numeric,
	new_expiration bigint,
	approved_by_beneficiary boolean,
	approved_by_nominee boolean
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_beneficiaries ADD CONSTRAINT miner_beneficiaries_pkey PRIMARY KEY (height, miner_id, state_root);
CREATE INDEX miner_beneficiaries_height_idx ON {{ .SchemaName | default "public"}}.miner_beneficiaries USING btree (height DESC);
`,
	)
}
