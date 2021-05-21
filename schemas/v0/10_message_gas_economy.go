package v0

func init() {
	up := batch(`
CREATE TABLE IF NOT EXISTS public.message_gas_economy (
	"state_root" text NOT NULL,
	"gas_limit_total" bigint NOT NULL,
	"gas_limit_unique_total" bigint NULL,
	"base_fee" double precision NOT NULL,
	"base_fee_change_log" double precision NOT NULL,
	"gas_fill_ratio" double precision NULL,
	"gas_capacity_ratio" double precision NULL,
	"gas_waste_ratio" double precision NULL,
	PRIMARY KEY ("state_root")
);
`)
	down := batch(`
DROP TABLE IF EXISTS public.message_gas_economy;
`)

	Patches.MustRegisterTx(up, down)
}
