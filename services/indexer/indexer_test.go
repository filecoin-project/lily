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

	t.Logf("collecting chain cids")
	cids, err := collectChainCids(node, head)
	require.NoError(t, err, "collect chain cids")

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
		var bs *blocks.BlockSynced
		for _, cid := range cids {
			exists, err := db.DB.Model(bs).Where("cid = ?", cid).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "cid: %s", cid)
		}
	})

	t.Run("block_headers", func(t *testing.T) {
		var bh *blocks.BlockHeader
		for _, cid := range cids {
			exists, err := db.DB.Model(bh).Where("cid = ?", cid).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "cid: %s", cid)
		}
	})

	t.Run("block_parents", func(t *testing.T) {
		var bp *blocks.BlockParent
		for _, cid := range cids {
			exists, err := db.DB.Model(bp).Where("block = ?", cid).Exists()
			require.NoError(t, err)
			assert.True(t, exists, "block: %s", cid)
		}
	})
}

// collectChainCids walks the chain to collect cids that should be indexed
func collectChainCids(n lens.API, ts *types.TipSet) ([]string, error) {
	var cids []string
	for _, c := range ts.Cids() {
		cids = append(cids, c.String())
	}

	for _, bh := range ts.Blocks() {
		if bh.Height == 0 {
			continue
		}

		parent, err := n.ChainGetTipSet(context.TODO(), types.NewTipSetKey(bh.Parents...))
		if err != nil {
			return nil, err
		}

		pcids, err := collectChainCids(n, parent)
		if err != nil {
			return nil, err
		}
		cids = append(cids, pcids...)

	}
	return cids, nil
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
