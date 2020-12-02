package indexer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	apitest "github.com/filecoin-project/lotus/api/test"
	"github.com/filecoin-project/lotus/build"
	types "github.com/filecoin-project/lotus/chain/types"
	nodetest "github.com/filecoin-project/lotus/node/test"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/builtin/verifreg"
	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
	"github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/testutil"
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
}

func TestChainHeadIndexer(t *testing.T) {
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
	nodes, sn := nodetest.RPCMockSbBuilder(t, apitest.OneFull, apitest.OneMiner)

	node := nodes[0]
	opener := testutil.NewAPIOpener(node)

	apitest.MineUntilBlock(ctx, t, node, sn[0], nil)

	blockIndexer := NewTipSetBlockIndexer(&storage.Database{DB: db})
	t.Logf("initializing indexer")
	idx := NewChainHeadIndexer(blockIndexer, opener, 0)

	newHeads, err := node.ChainNotify(ctx)
	require.NoError(t, err, "chain notify")

	t.Logf("indexing chain")
	nh := <-newHeads

	var bhs blockHeaderList
	for _, head := range nh {
		bhs = append(bhs, head.Val.Blocks()...)
	}

	cids := bhs.Cids()
	rounds := bhs.Rounds()
	chainHead, err := node.ChainHead(ctx)
	require.NoError(t, err, "ChainHead")

	tipSetKeys := []string{chainHead.Key().String()}

	err = idx.index(ctx, nh)
	require.NoError(t, err, "index")

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

func collectTipSetKeys(n lens.API, ts *types.TipSet) ([]string, error) {
	keys := []string{ts.Key().String()}

	for ts.Height() > 0 {
		parent, err := n.ChainGetTipSet(context.TODO(), ts.Parents())
		if err != nil {
			return nil, err
		}
		keys = append(keys, parent.Key().String())
		ts = parent
	}

	return keys, nil
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

	_, err = db.Exec(`TRUNCATE TABLE visor_processing_tipsets`)
	require.NoError(tb, err, "visor_processing_tipsets")

	return nil
}
