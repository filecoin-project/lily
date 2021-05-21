package v0

// Schema version 17 changes materialized views for chain_visualizer to simple views

func init() {
	up := batch(`
DROP MATERIALIZED VIEW IF EXISTS chain_visualizer_blocks_view;
DROP MATERIALIZED VIEW IF EXISTS chain_visualizer_blocks_with_parents_view;
DROP MATERIALIZED VIEW IF EXISTS chain_visualizer_orphans_view;
DROP MATERIALIZED VIEW IF EXISTS chain_visualizer_chain_data_view;

CREATE VIEW chain_visualizer_chain_data_view AS
	SELECT
		main_block.cid AS block,
		bp.parent AS parent,
		main_block.miner,
		main_block.height,
		main_block.parent_weight AS parentweight,
		main_block.timestamp,
		main_block.parent_state_root AS parentstateroot,
		parent_block.timestamp AS parenttimestamp,
		parent_block.height AS parentheight,
		mp.raw_bytes_power AS parentpower,
		synced.synced_at AS syncedtimestamp,
		(SELECT COUNT(*) FROM block_messages WHERE block_messages.block = main_block.cid) AS messages
	FROM
		block_headers main_block
	LEFT JOIN
		block_parents bp ON bp.block = main_block.cid
	LEFT JOIN
		block_headers parent_block ON parent_block.cid = bp.parent
	LEFT JOIN
		blocks_synced synced ON synced.cid = main_block.cid
	LEFT JOIN
		miner_power mp ON main_block.parent_state_root = mp.state_root
;

CREATE VIEW chain_visualizer_orphans_view AS
	SELECT
		block_headers.cid AS block,
		block_headers.miner,
		block_headers.height,
		block_headers.parent_weight AS parentweight,
		block_headers.timestamp,
		block_headers.parent_state_root AS parentstateroot,
		block_parents.parent AS parent
	FROM
		block_headers
	LEFT JOIN
		block_parents ON block_headers.cid = block_parents.parent
	WHERE
		block_parents.block IS NULL
;

CREATE VIEW chain_visualizer_blocks_with_parents_view AS
	SELECT
		block,
		parent,
		b.miner,
		b.height,
		b.timestamp
	FROM
		block_parents
	INNER JOIN
		block_headers b ON block_parents.block = b.cid
;

CREATE VIEW chain_visualizer_blocks_view AS
	SELECT * FROM block_headers
;
`)

	down := batch(`
DROP VIEW IF EXISTS chain_visualizer_blocks_view;
DROP VIEW IF EXISTS chain_visualizer_blocks_with_parents_view;
DROP VIEW IF EXISTS chain_visualizer_orphans_view;
DROP VIEW IF EXISTS chain_visualizer_chain_data_view;

CREATE MATERIALIZED VIEW IF NOT EXISTS chain_visualizer_chain_data_view AS
	SELECT
		main_block.cid AS block,
		bp.parent AS parent,
		main_block.miner,
		main_block.height,
		main_block.parent_weight AS parentweight,
		main_block.timestamp,
		main_block.parent_state_root AS parentstateroot,
		parent_block.timestamp AS parenttimestamp,
		parent_block.height AS parentheight,
		mp.raw_bytes_power AS parentpower,
		synced.synced_at AS syncedtimestamp,
		(SELECT COUNT(*) FROM block_messages WHERE block_messages.block = main_block.cid) AS messages
	FROM
		block_headers main_block
	LEFT JOIN
		block_parents bp ON bp.block = main_block.cid
	LEFT JOIN
		block_headers parent_block ON parent_block.cid = bp.parent
	LEFT JOIN
		blocks_synced synced ON synced.cid = main_block.cid
	LEFT JOIN
		miner_power mp ON main_block.parent_state_root = mp.state_root
WITH NO DATA;

CREATE MATERIALIZED VIEW IF NOT EXISTS chain_visualizer_orphans_view AS
	SELECT
		block_headers.cid AS block,
		block_headers.miner,
		block_headers.height,
		block_headers.parent_weight AS parentweight,
		block_headers.timestamp,
		block_headers.parent_state_root AS parentstateroot,
		block_parents.parent AS parent
	FROM
		block_headers
	LEFT JOIN
		block_parents ON block_headers.cid = block_parents.parent
	WHERE
		block_parents.block IS NULL
WITH NO DATA;

CREATE MATERIALIZED VIEW IF NOT EXISTS chain_visualizer_blocks_with_parents_view AS
	SELECT
		block,
		parent,
		b.miner,
		b.height,
		b.timestamp
	FROM
		block_parents
	INNER JOIN
		block_headers b ON block_parents.block = b.cid
WITH NO DATA;

CREATE MATERIALIZED VIEW IF NOT EXISTS chain_visualizer_blocks_view AS
	SELECT * FROM block_headers
WITH NO DATA;
`)

	Patches.MustRegisterTx(up, down)
}
