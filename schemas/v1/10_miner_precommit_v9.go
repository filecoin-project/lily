package v1

// Schema patch 10 adds miner precommitinfos v9

func init() {
	patches.Register(
		10,
		`
CREATE TABLE {{ .SchemaName | default "public"}}.miner_pre_commit_infos_v9 (
    height bigint NOT NULL,
    state_root text NOT NULL,
    miner_id text NOT NULL,
    sector_id bigint NOT NULL,
   	pre_commit_deposit numeric NOT NULL,
    pre_commit_epoch bigint NOT NULL,
    sealed_cid text NOT NULL,
    seal_rand_epoch bigint NOT NULL,
    expiration_epoch bigint NOT NULL,
    deal_ids bigint[],
    unsealed_cid text
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.miner_pre_commit_infos_v9 ADD CONSTRAINT miner_pre_commit_infos_v9_pkey PRIMARY KEY (height, miner_id, sector_id, state_root);
CREATE INDEX miner_pre_commit_infos_v9_height_idx ON {{ .SchemaName | default "public"}}.miner_pre_commit_infos_v9 USING btree (height DESC);
`,
	)
}
