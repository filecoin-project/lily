package indexer

import (
	"context"
	"os"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	apitest "github.com/filecoin-project/lotus/api/test"
	"github.com/filecoin-project/lotus/build"
	types "github.com/filecoin-project/lotus/chain/types"
	nodetest "github.com/filecoin-project/lotus/node/test"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/builtin/verifreg"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model/blocks"
	"github.com/filecoin-project/sentinel-visor/storage"
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

type nodeWrapper struct {
	apitest.TestNode
}

func (nodeWrapper) Store() adt.Store {
	panic("not supported")
}

func TestIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing specified but this test requires external dependencies")
	}

	db, err := storage.NewDatabase(context.Background(), "postgres://postgres:password@localhost:5432/postgres?sslmode=disable")
	require.NoError(t, err, "connecting to database")

	t.Logf("truncating database tables")
	err = truncateBlockTables(db)
	require.NoError(t, err, "truncating tables")

	t.Logf("preparing chain")
	ctx := context.Background()
	nodes, sn := nodetest.Builder(t, 1, apitest.OneMiner)
	node := nodeWrapper{TestNode: nodes[0]}

	apitest.MineUntilBlock(ctx, t, nodes[0], sn[0], nil)

	head, err := node.ChainHead(ctx)
	require.NoError(t, err, "chain head")

	t.Logf("collecting chain blocks")
	bhs, err := collectBlockHeaders(node, head)
	require.NoError(t, err, "collect chain blocks")

	cids := bhs.Cids()
	rounds := bhs.Rounds()

	t.Logf("initializing indexer")
	idx := NewIndexer(db, node)
	err = idx.InitHandler(ctx)
	require.NoError(t, err, "init handler")

	newHeads, err := node.ChainNotify(ctx)
	require.NoError(t, err, "chain notify")

	t.Logf("indexing chain")
	nh := <-newHeads
	err = idx.index(ctx, nh)
	require.NoError(t, err, "index")

	t.Run("blocks_synced", func(t *testing.T) {
		var count int
		_, err := db.DB.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM blocks_synced`)
		require.NoError(t, err)
		assert.Equal(t, len(cids), count)

		var m *blocks.BlockSynced
		for _, cid := range cids {
			exists, err := db.DB.Model(m).Where("cid = ?", cid).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "cid: %s", cid)
		}
	})

	t.Run("block_headers", func(t *testing.T) {
		var count int
		_, err := db.DB.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_headers`)
		require.NoError(t, err)
		assert.Equal(t, len(cids), count)

		var m *blocks.BlockHeader
		for _, cid := range cids {
			exists, err := db.DB.Model(m).Where("cid = ?", cid).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "cid: %s", cid)
		}
	})

	t.Run("block_parents", func(t *testing.T) {
		var count int
		_, err := db.DB.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_parents`)
		require.NoError(t, err)
		assert.Equal(t, len(cids), count)

		var m *blocks.BlockParent
		for _, cid := range cids {
			exists, err := db.DB.Model(m).Where("block = ?", cid).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "block: %s", cid)
		}
	})

	t.Run("drand_entries", func(t *testing.T) {
		var count int
		_, err := db.DB.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM drand_entries`)
		require.NoError(t, err)
		assert.Equal(t, len(rounds), count)

		var m *blocks.DrandEntrie
		for _, round := range rounds {
			exists, err := db.DB.Model(m).Where("round = ?", round).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "round: %d", round)
		}
	})

	t.Run("drand_block_entries", func(t *testing.T) {
		var count int
		_, err := db.DB.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM drand_block_entries`)
		require.NoError(t, err)
		assert.Equal(t, len(rounds), count)

		var m *blocks.DrandBlockEntrie
		for _, round := range rounds {
			exists, err := db.DB.Model(m).Where("round = ?", round).Exists()
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
func truncateBlockTables(db *storage.Database) error {
	if _, err := db.DB.Exec(`TRUNCATE TABLE blocks_synced`); err != nil {
		return err
	}

	if _, err := db.DB.Exec(`TRUNCATE TABLE block_headers`); err != nil {
		return err
	}

	if _, err := db.DB.Exec(`TRUNCATE TABLE block_parents`); err != nil {
		return err
	}

	if _, err := db.DB.Exec(`TRUNCATE TABLE drand_entries`); err != nil {
		return err
	}

	if _, err := db.DB.Exec(`TRUNCATE TABLE drand_block_entries`); err != nil {
		return err
	}

	return nil
}
