package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 28 adds multisig_approvals

func init() {
	up := batch(`
	CREATE TABLE IF NOT EXISTS "multisig_approvals" (
		"height"           bigint  not null,
		"state_root"       text    not null,
        "multisig_id"      text    not null,
        "message"          text    not null,
        "method"           bigint  not null,
        "approver"         text    not null,
        "threshold"        bigint  not null,
        "initial_balance"  numeric not null,
        "gas_used"         bigint  not null,
        "transaction_id"   bigint  not null,
        "to"               text    not null,
        "value"            numeric not null,
        "signers"          jsonb  not null,
        PRIMARY KEY ("height", "state_root", "multisig_id")
	);

	-- Chunked per 30 days (86400 epochs)
	SELECT create_hypertable(
		'multisig_approvals',
		'height',
		chunk_time_interval => 86400,
		if_not_exists => TRUE
	);
`)

	down := batch(`
	DROP TABLE IF EXISTS public.multisig_approvals;
`)

	migrations.MustRegisterTx(up, down)
}
