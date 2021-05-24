package v0

// Schema version 18 adds multisig transactions

func init() {
	up := batch(`
	CREATE TABLE IF NOT EXISTS "multisig_transactions" (
		"height" bigint not null,
		"multisig_id" text not null,
		"state_root" text not null,
		"transaction_id" bigint not null,

		"to" text not null,
		"value" text not null,
		"method" bigint not null,
		"params" bytea not null,
		"approved" jsonb not null,
		PRIMARY KEY ("height", "state_root", "multisig_id", "transaction_id")
	);
`)

	down := batch(`
	DROP TABLE IF EXISTS public.multisig_transactions;
`)

	Patches.MustRegisterTx(up, down)
}
