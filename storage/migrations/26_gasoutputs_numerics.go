package migrations

import "github.com/go-pg/migrations/v8"

// Schema version 26 converts derived_gas_outputs text columns to numerics

func init() {
	up := batch(`
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN value TYPE numeric USING (value::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN gas_fee_cap TYPE numeric USING (gas_fee_cap::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN gas_premium TYPE numeric USING (gas_premium::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN parent_base_fee TYPE numeric USING (parent_base_fee::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN base_fee_burn TYPE numeric USING (base_fee_burn::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN over_estimation_burn TYPE numeric USING (over_estimation_burn::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN miner_penalty TYPE numeric USING (miner_penalty::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN miner_tip TYPE numeric USING (miner_tip::numeric);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN refund TYPE numeric USING (refund::numeric);
`)
	down := batch(`
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN value TYPE text USING (value::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN gas_fee_cap TYPE text USING (gas_fee_cap::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN gas_premium TYPE text USING (gas_premium::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN parent_base_fee TYPE text USING (parent_base_fee::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN base_fee_burn TYPE text USING (base_fee_burn::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN over_estimation_burn TYPE text USING (over_estimation_burn::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN miner_penalty TYPE text USING (miner_penalty::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN miner_tip TYPE text USING (miner_tip::text);
	ALTER TABLE public.derived_gas_outputs ALTER COLUMN refund TYPE text USING (refund::text);
`)

	migrations.MustRegisterTx(up, down)
}
