package v1

func init() {
	patches.Register(
		40,
		`
		ALTER TABLE {{ .SchemaName | default "public"}}.chain_economics DROP COLUMN IF NOT EXISTS locked_fil_v2;

		CREATE TABLE {{ .SchemaName | default "public"}}.chain_economics_v2 (
			height bigint NOT NULL,
			parent_state_root text NOT NULL,
			circulating_fil_v2 numeric NOT NULL,
			vested_fil numeric NOT NULL,
			mined_fil numeric NOT NULL,
			burnt_fil numeric NOT NULL,
			locked_fil_v2 numeric NOT NULL,
			fil_reserve_disbursed numeric NOT NULL
		);
		ALTER TABLE ONLY {{ .SchemaName | default "public"}}.chain_economics_v2 ADD CONSTRAINT chain_economics_pk PRIMARY KEY (height, parent_state_root);
		`,
	)
}
