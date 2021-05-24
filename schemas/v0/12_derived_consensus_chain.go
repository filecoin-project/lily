package v0

// Schema version 8 adds view over derived_consensus_chain_view

func init() {
	up := batch(`
-- drop old mis-named view if it exists
DROP MATERIALIZED VIEW IF EXISTS derived_conensus_chain_view;

CREATE MATERIALIZED VIEW IF NOT EXISTS derived_consensus_chain_view AS
WITH RECURSIVE consensus_chain AS (
	SELECT
		b.cid,
		b.height,
		b.miner,
		b.timestamp,
		b.parent_state_root,
		b.win_count
	FROM block_headers b
	WHERE b.parent_state_root = (SELECT parent_state_root FROM block_headers ORDER BY height desc, parent_weight DESC LIMIT 1)
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
) SELECT * FROM consensus_chain
WITH NO DATA;
`)

	down := batch(`
DROP MATERIALIZED VIEW IF EXISTS derived_consensus_chain_view;
	`)

	Patches.MustRegisterTx(up, down)
}
