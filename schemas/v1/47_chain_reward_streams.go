package v1

func init() {
	patches.Register(
		47,
		`
		CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.chain_reward_streams (
			height bigint NOT NULL,
			state_root text NOT NULL,
			total_minted_reward numeric NOT NULL,
			total_burn_minted numeric NOT NULL,
			total_explicit_minted numeric NOT NULL,
			burn_weight numeric NOT NULL,
			consensus_weight numeric NOT NULL,
			service_weight numeric NOT NULL,
			consensus_v_start numeric NOT NULL,
			consensus_slope numeric NOT NULL,
			consensus_t_start bigint NOT NULL,
			consensus_floor numeric NOT NULL,
			consensus_cap numeric NOT NULL,
			service_v_start numeric NOT NULL,
			service_slope numeric NOT NULL,
			service_t_start bigint NOT NULL,
			service_floor numeric NOT NULL,
			service_cap numeric NOT NULL
		);
		ALTER TABLE ONLY {{ .SchemaName | default "public"}}.chain_reward_streams ADD CONSTRAINT chain_reward_streams_pk PRIMARY KEY (height, state_root);

		CREATE INDEX IF NOT EXISTS chain_reward_streams_height_idx ON {{ .SchemaName | default "public"}}.chain_reward_streams USING btree (height DESC);

		COMMENT ON TABLE {{ .SchemaName | default "public"}}.chain_reward_streams IS 'Per-epoch FIP-0118 block-reward stream weights and minted totals from the reward actor. Weights are DENOM (1e18) fixed-point integers.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.height IS 'Epoch this stream summary applies to.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.state_root IS 'CID of the parent state root.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.total_minted_reward IS 'Total FIL (attoFIL) minted through block rewards, across all streams.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.total_burn_minted IS 'Cumulative block-reward residual (attoFIL) sent to the burn actor.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.total_explicit_minted IS 'Cumulative block reward (attoFIL) accrued to explicit streams.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.burn_weight IS 'Burn stream weight w0 at this epoch, in DENOM fixed point. Residual over every live stream.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.consensus_weight IS 'Consensus stream weight w1 (stream id 1) at this epoch, in DENOM fixed point.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.service_weight IS 'Service stream weight w2 (stream id 2) at this epoch, in DENOM fixed point.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.consensus_v_start IS 'Consensus weight at consensus_t_start, in DENOM fixed point.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.consensus_slope IS 'Consensus weight change per epoch, in DENOM fixed point. Negative while the ramp declines.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.consensus_t_start IS 'Epoch at which consensus_v_start applies.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.consensus_floor IS 'Consensus weight lower clamp, in DENOM fixed point.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.consensus_cap IS 'Consensus weight upper clamp, in DENOM fixed point.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.service_v_start IS 'Service weight at service_t_start, in DENOM fixed point.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.service_slope IS 'Service weight change per epoch, in DENOM fixed point.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.service_t_start IS 'Epoch at which service_v_start applies.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.service_floor IS 'Service weight lower clamp, in DENOM fixed point.';
		COMMENT ON COLUMN {{ .SchemaName | default "public"}}.chain_reward_streams.service_cap IS 'Service weight upper clamp, in DENOM fixed point.';
		`,
	)
}
