package chain

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/build"
	types "github.com/filecoin-project/lotus/chain/types"
	itestkit "github.com/filecoin-project/lotus/itests/kit"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/builtin/verifreg"
	"github.com/go-pg/pg/v10"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/task"
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
	err = truncateBlockTables(t, db)
	require.NoError(t, err, "truncating tables")

	t.Logf("preparing chain")
	full, miner, _ := itestkit.EnsembleMinimal(t, itestkit.MockProofs())

	nodeAPI := testutil.NewAPIWrapper(full)

	strg, err := storage.NewDatabaseFromDB(ctx, db, "public")
	require.NoError(t, err, "NewDatabaseFromDB")

	taskAPI, err := task.NewTaskAPI(nodeAPI)
	require.NoError(t, err)
	tsIndexer, err := NewTipSetIndexer(taskAPI, strg, builtin.EpochDurationSeconds*time.Second, t.Name(), []string{BlocksTask})
	require.NoError(t, err, "NewTipSetIndexer")
	t.Logf("initializing indexer")
	idx := NewWatcher(tsIndexer, NullHeadNotifier{}, NewTipSetCache(0))

	newHeads, err := full.ChainNotify(ctx)
	require.NoError(t, err, "chain notify")

	bm := itestkit.NewBlockMiner(t, miner)
	t.Logf("mining first block")
	bm.MineUntilBlock(ctx, full, nil)

	first := <-newHeads

	var bhs blockHeaderList
	for _, head := range first {
		bhs = append(bhs, head.Val.Blocks()...)
	}

	cids := bhs.Cids()
	rounds := bhs.Rounds()

	t.Logf("indexing first tipset")
	for _, hc := range first {
		he := &HeadEvent{Type: hc.Type, TipSet: hc.Val}
		err = idx.index(ctx, he)
		require.NoError(t, err, "index")
	}

	t.Logf("mining second block")
	bm.MineUntilBlock(ctx, full, nil)

	second := <-newHeads
	t.Logf("indexing second tipset")
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

type blockHeaderList []*types.BlockHeader

func (b blockHeaderList) Cids() []string {
	var cids []string
	for _, bh := range b {
		cids = append(cids, bh.Cid().String())
	}
	return cids
}

func (b blockHeaderList) Rounds() []uint64 {
	var rounds []uint64
	for _, bh := range b {
		for _, ent := range bh.BeaconEntries {
			rounds = append(rounds, ent.Round)
		}
	}

	return rounds
}

// collectBlockHeaders walks the chain to collect blocks that should be indexed
func collectBlockHeaders(n lens.API, ts *types.TipSet) (blockHeaderList, error) {
	blocks := ts.Blocks()

	for _, bh := range ts.Blocks() {
		if bh.Height == 0 {
			continue
		}

		parent, err := n.ChainGetTipSet(context.TODO(), types.NewTipSetKey(bh.Parents...))
		if err != nil {
			return nil, err
		}

		pblocks, err := collectBlockHeaders(n, parent)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, pblocks...)

	}
	return blocks, nil
}

// truncateBlockTables ensures the indexing tables are empty
func truncateBlockTables(tb testing.TB, db *pg.DB) error {
	_, err := db.Exec(`TRUNCATE TABLE block_headers`)
	require.NoError(tb, err, "block_headers")

	_, err = db.Exec(`TRUNCATE TABLE block_parents`)
	require.NoError(tb, err, "block_parents")

	_, err = db.Exec(`TRUNCATE TABLE drand_block_entries`)
	require.NoError(tb, err, "drand_block_entries")

	return nil
}

type NullHeadNotifier struct{}

func (NullHeadNotifier) HeadEvents() <-chan *HeadEvent { return nil }
func (NullHeadNotifier) Err() error                    { return nil }
