package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 20 removes unused binary data

func init() {
	up := batch(`
	DROP TABLE IF EXISTS public.drand_entries;

	-- view depends on ticket and election_proof
	DROP VIEW IF EXISTS chain_visualizer_blocks_view;
	ALTER TABLE public.block_headers DROP COLUMN ticket;
	ALTER TABLE public.block_headers DROP COLUMN election_proof;

	CREATE VIEW chain_visualizer_blocks_view AS
		SELECT * FROM block_headers
	;

	ALTER TABLE public.messages DROP COLUMN params;

	ALTER TABLE public.receipts DROP COLUMN return;
`)

	down := batch(`
	CREATE TABLE public.drand_entries (
		round bigint NOT NULL,
		data bytea NOT NULL
	);

	-- view depends on ticket and election_proof
	DROP VIEW IF EXISTS chain_visualizer_blocks_view;
	ALTER TABLE public.block_headers ADD COLUMN ticket bytea;
	ALTER TABLE public.block_headers ADD COLUMN election_proof bytea;
	CREATE VIEW chain_visualizer_blocks_view AS
		SELECT * FROM block_headers
	;

	ALTER TABLE public.messages ADD COLUMN params bytea;

	ALTER TABLE public.receipts ADD COLUMN return bytea;
`)

	migrations.MustRegisterTx(up, down)
}
