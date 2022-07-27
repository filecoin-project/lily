package v1

func init() {
	patches.Register(
		8,
		`
------------------------------------------------------------------
-- Name: miner_sector_events_event_idx
-- Model: miner.MinerSectorEvent
-- ----------------------------------------------------------------
CREATE INDEX miner_sector_events_event_idx ON {{ .SchemaName | default "public"}}.miner_sector_events
	USING btree(event) INCLUDE(height, miner_id) WITH(timescaledb.transaction_per_chunk);
`,
	)
}
