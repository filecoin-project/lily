package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 9 adds a table for processing stats and indexes over the visor processing tables

func init() {
	up := batch(`
CREATE TABLE IF NOT EXISTS public.visor_processing_stats (
	"recorded_at" timestamptz NOT NULL,
	"measure" text NOT NULL,
	"value" bigint NOT NULL,
	PRIMARY KEY ("recorded_at","measure")
);
`)

	down := batch(`
DROP TABLE IF EXISTS public.visor_processing_stats;
`)
	migrations.MustRegisterTx(up, down)

	deferredUp := batch(`
CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_tipsets_statechange_idx" ON public.visor_processing_tipsets USING BTREE (statechange_completed_at, statechange_claimed_until);
CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_tipsets_message_idx"     ON public.visor_processing_tipsets USING BTREE (message_completed_at, message_claimed_until);
CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_tipsets_economics_idx"   ON public.visor_processing_tipsets USING BTREE (economics_completed_at, economics_claimed_until);
CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_tipsets_height_idx"      ON public.visor_processing_tipsets USING BTREE (height DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_messages_gas_outputs_idx" ON public.visor_processing_messages USING BTREE (gas_outputs_completed_at, gas_outputs_claimed_until);
CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_messages_height_idx"      ON public.visor_processing_messages USING BTREE (height DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_actors_completed_idx" ON public.visor_processing_actors USING BTREE (completed_at, claimed_until);
CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_actors_code_idx"      ON public.visor_processing_actors USING HASH (code);
CREATE INDEX CONCURRENTLY IF NOT EXISTS "visor_processing_actors_height_idx"    ON public.visor_processing_actors USING BTREE (height DESC);
`)

	deferredDown := batch(`
DROP INDEX IF EXISTS visor_processing_tipsets_statechange_idx;
DROP INDEX IF EXISTS visor_processing_tipsets_message_idx;
DROP INDEX IF EXISTS visor_processing_tipsets_economics_idx;
DROP INDEX IF EXISTS visor_processing_tipsets_height_idx;

DROP INDEX IF EXISTS visor_processing_messages_gas_outputs_idx;
DROP INDEX IF EXISTS visor_processing_messages_height_idx;

DROP INDEX IF EXISTS visor_processing_actors_completed_idx;
DROP INDEX IF EXISTS visor_processing_actors_code_idx;
DROP INDEX IF EXISTS visor_processing_actors_height_idx;
`)

	MustRegisterDeferred(deferredUp, deferredDown)
}
