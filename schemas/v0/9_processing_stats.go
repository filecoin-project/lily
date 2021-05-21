package v0

// Schema version 9 adds a table for processing stats and indexes over the visor processing tables

func init() {
	up := batch(`
CREATE TABLE IF NOT EXISTS public.visor_processing_stats (
	"recorded_at" timestamptz NOT NULL,
	"measure" text NOT NULL,
	"tag" text NOT NULL,
	"value" bigint NOT NULL,
	PRIMARY KEY ("recorded_at","measure","tag")
);


-- Convert visor_processing_stats to a hypertable partitioned by recorded_at
-- 1 tipset per epoch
-- One chunk per day, 3600 sets of stats per chunk, a set may contain 30-40 measurements
SELECT create_hypertable(
	'visor_processing_stats',
	'recorded_at',
	chunk_time_interval => INTERVAL '1 day',
	if_not_exists => TRUE
);

`)

	down := batch(`
DROP TABLE IF EXISTS public.visor_processing_stats;

`)
	Patches.MustRegisterTx(up, down)
}
