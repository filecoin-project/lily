package migrations

import "github.com/go-pg/migrations/v8"

// Schema version 26 removes unused blocks_synced table;

func init() {
	up := batch(`
	DROP TABLE IF EXISTS public.blocks_synced;
`)

	down := batch(`
CREATE TABLE public.blocks_synced (
    cid text NOT NULL,
    synced_at integer NOT NULL,
    processed_at integer
);
ALTER TABLE ONLY public.blocks_synced
    ADD CONSTRAINT blocks_synced_pk PRIMARY KEY (cid);
CREATE UNIQUE INDEX blocks_synced_cid_uindex ON public.blocks_synced USING btree (cid, processed_at);
`)

	migrations.MustRegisterTx(up, down)
}
