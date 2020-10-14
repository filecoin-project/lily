package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 8 adds view over derived_conensus_chain_view

func init() {
	up := batch(`
CREATE MATERIALIZED VIEW IF NOT EXISTS derived_conensus_chain_view AS
WITH RECURSIVE consensus_chain AS (
	SELECT
		b.cid,
		b.height,
		b.miner,
		b.timestamp,
		b.parent_state_root,
		b.win_count
	FROM block_headers b
	WHERE b.parent_state_root = (select parent_state_root from block_headers order by height desc, parent_weight desc limit 1)
	UNION
	SELECT
		p.cid,
		p.height,
		p.miner,
		p.timestamp,
		p.parent_state_root,
		p.win_count
	FROM block_headers p
	INNER JOIN block_parents pb ON p.cid = pb.parent
	INNER JOIN consensus_chain c ON c.cid = pb.block
) select * from consensus_chain
WITH NO DATA;
`)

	down := batch(`
DROP MATERIALIZED VIEW IF EXISTS derived_conensus_chain_view;
	`)

	migrations.MustRegisterTx(up, down)
}
