package indexer

import (
	"context"
	"testing"
	"time"

	apitest "github.com/filecoin-project/lotus/api/test"
	nodetest "github.com/filecoin-project/lotus/node/test"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/lens/lotus"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
	"github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/testutil"
)

func TestChainHistoryIndexer(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer cleanup()

	t.Logf("truncating database tables")
	err = truncateBlockTables(t, db)
	require.NoError(t, err, "truncating tables")

	t.Logf("preparing chain")
	nodes, sn := nodetest.Builder(t, apitest.DefaultFullOpts(1), apitest.OneMiner)
	cs, err := lotus.NewCacheCtxStore(ctx, nodes[0], 5)
	require.NoError(t, err, "cache store")
	node := lotus.NewAPIWrapper(nodes[0], cs)

	apitest.MineUntilBlock(ctx, t, nodes[0], sn[0], nil)

	head, err := node.ChainHead(ctx)
	require.NoError(t, err, "chain head")

	t.Logf("collecting chain blocks")
	bhs, err := collectBlockHeaders(node, head)
	require.NoError(t, err, "collect chain blocks")

	tipSetKeys, err := collectTipSetKeys(node, head)
	require.NoError(t, err, "collect chain blocks")

	cids := bhs.Cids()
	rounds := bhs.Rounds()

	d := &storage.Database{DB: db}
	t.Logf("initializing indexer")
	idx := NewChainHistoryIndexer(d, lens.API(node), 1)

	t.Logf("indexing chain")
	err = idx.WalkChain(ctx, int64(head.Height()))
	require.NoError(t, err, "WalkChain")

	t.Run("block_headers", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_headers`)
		require.NoError(t, err)
		assert.Equal(t, len(cids), count)

		var m *blocks.BlockHeader
		for _, cid := range cids {
			exists, err := db.Model(m).Where("cid = ?", cid).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "cid: %s", cid)
		}
	})

	t.Run("block_parents", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_parents`)
		require.NoError(t, err)
		assert.Equal(t, len(cids), count)

		var m *blocks.BlockParent
		for _, cid := range cids {
			exists, err := db.Model(m).Where("block = ?", cid).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "block: %s", cid)
		}
	})

	t.Run("drand_entries", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM drand_entries`)
		require.NoError(t, err)
		assert.Equal(t, len(rounds), count)

		var m *blocks.DrandEntrie
		for _, round := range rounds {
			exists, err := db.Model(m).Where("round = ?", round).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "round: %d", round)
		}
	})

	t.Run("drand_block_entries", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM drand_block_entries`)
		require.NoError(t, err)
		assert.Equal(t, len(rounds), count)

		var m *blocks.DrandBlockEntrie
		for _, round := range rounds {
			exists, err := db.Model(m).Where("round = ?", round).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "round: %d", round)
		}
	})

	t.Run("visor_processing_tipsets", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM visor_processing_tipsets`)
		require.NoError(t, err)
		assert.Equal(t, len(tipSetKeys), count)

		var m *visor.ProcessingTipSet
		for _, tsk := range tipSetKeys {
			exists, err := db.Model(m).Where("tip_set = ?", tsk).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "tsk: %s", tsk)
		}
	})
}
