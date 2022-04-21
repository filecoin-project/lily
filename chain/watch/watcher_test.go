package watch

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/build"
	itestkit "github.com/filecoin-project/lotus/itests/kit"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/builtin/verifreg"
	"github.com/gammazero/workerpool"
	"github.com/go-pg/pg/v10"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/cache"
	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/chain/indexer/integrated"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model/blocks"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lily/testutil"
)

func init() {
	build.InsecurePoStValidation = true
	err := os.Setenv("TRUST_PARAMS", "1")
	if err != nil {
		panic(err)
	}
	miner.SupportedProofTypes = map[abi.RegisteredSealProof]struct{}{
		abi.RegisteredSealProof_StackedDrg2KiBV1: {},
	}
	power.ConsensusMinerMinPower = big.NewInt(2048)
	verifreg.MinVerifiedDealSize = big.NewInt(256)

	logging.SetLogLevel("*", "ERROR")
	logging.SetLogLevelRegex("visor/.+", "DEBUG")
}

func TestWatcher(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	t.Logf("truncating database tables")
	err = testutil.TruncateBlockTables(t, db)
	require.NoError(t, err, "truncating tables")

	t.Logf("preparing chain")
	full, miner, _ := itestkit.EnsembleMinimal(t, itestkit.MockProofs())

	nodeAPI := testutil.NewAPIWrapper(full)

	strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
	require.NoError(t, err, "NewDatabaseFromDB")

	taskAPI, err := datasource.NewDataSource(nodeAPI)
	require.NoError(t, err)
	im, err := integrated.NewManager(taskAPI, strg, t.Name(), integrated.WithWindow(builtin.EpochDurationSeconds*time.Second))
	require.NoError(t, err, "NewManager")
	t.Logf("initializing indexer")
	idx := NewWatcher(nil, im, t.Name(), WithConfidence(0), WithConcurrentWorkers(1), WithBufferSize(5), WithTasks(tasktype.BlocksTask))
	idx.cache = cache.NewTipSetCache(0)
	// the watchers worker pool and cache are initialized in its Run method, since we don't call that here initialize them now.
	idx.pool = workerpool.New(1)

	newHeads, err := full.ChainNotify(ctx)
	require.NoError(t, err, "chain notify")

	bm := itestkit.NewBlockMiner(t, miner)
	t.Logf("mining first block")
	bm.MineUntilBlock(ctx, full, nil)
	first := <-newHeads
	var bhs testutil.BlockHeaderList
	for _, head := range first {
		bhs = append(bhs, head.Val.Blocks()...)
	}

	t.Logf("mining second block")
	bm.MineUntilBlock(ctx, full, nil)
	second := <-newHeads
	for _, head := range second {
		bhs = append(bhs, head.Val.Blocks()...)
	}

	cids := bhs.Cids()
	rounds := bhs.Rounds()

	// the `current` tipset being indexed is always the parent of the passed tipset.
	t.Logf("indexing first tipset")
	// here we will index first parents
	for _, hc := range first {
		he := &HeadEvent{Type: hc.Type, TipSet: hc.Val}
		err = idx.index(ctx, he)
		require.NoError(t, err, "index")
	}

	t.Logf("indexing second tipset")
	// here we will index second parents (so first)
	for _, hc := range second {
		he := &HeadEvent{Type: hc.Type, TipSet: hc.Val}
		err = idx.index(ctx, he)
		require.NoError(t, err, "index")
	}

	// TODO NewTipSetIndexer runs its processors in their own go routines (started when TipSet() is called)
	// this causes this test to behave nondeterministicly so we sleep here to ensure all async jobs
	// have completed before asserting results
	time.Sleep(time.Second * 3)

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
}
