package walk

import (
	"context"
	"testing"
	"time"

	itestkit "github.com/filecoin-project/lotus/itests/kit"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/datasource"
	"github.com/filecoin-project/lily/chain/indexer/integrated"
	"github.com/filecoin-project/lily/chain/indexer/integrated/tipset"
	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model/blocks"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lily/testutil"
)

func TestWalker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	t.Logf("preparing chain")

	logging.SetAllLoggers(logging.LevelInfo)
	full, miner, _ := itestkit.EnsembleMinimal(t, itestkit.MockProofs())

	nodeAPI := testutil.NewAPIWrapper(full)

	bm := itestkit.NewBlockMiner(t, miner)
	bm.MineUntilBlock(ctx, full, nil)

	head, err := full.ChainHead(ctx)
	require.NoError(t, err, "chain head")
	t.Logf("got chain head: %v height %d", head, head.Height())

	t.Logf("collecting chain blocks from tipset before head")
	bhs, err := testutil.CollectBlockHeaders(nodeAPI, head)
	require.NoError(t, err, "collect chain blocks")

	t.Logf("collected chain blocks: %v heights %v", bhs.Cids(), bhs.Heights())
	expectedCIDs := bhs.Cids()
	expectedRounds := bhs.Rounds()

	strg := storage.NewMemStorageLatest()

	taskAPI, err := datasource.NewDataSource(nodeAPI)
	require.NoError(t, err)

	im, err := integrated.NewManager(strg, tipset.NewBuilder(taskAPI, t.Name()), integrated.WithWindow(builtin.EpochDurationSeconds*time.Second))
	require.NoError(t, err, "NewManager")

	t.Logf("initializing indexer")
	idx := NewWalker(im, nodeAPI, t.Name(), []string{tasktype.BlocksTask}, 0, int64(head.Height()))

	t.Logf("indexing chain")
	err = idx.WalkChain(ctx, nodeAPI, head)
	require.NoError(t, err, "WalkChain")

	t.Run("block_headers", func(t *testing.T) {
		// expected count
		assert.Equal(t, len(expectedCIDs), len(strg.Data["block_headers"]))

		// models exist
		actualHeaders := make(map[string]struct{})
		for _, bh := range strg.Data["block_headers"] {
			actualHeaders[bh.(*blocks.BlockHeader).Cid] = struct{}{}
		}
		for _, cid := range expectedCIDs {
			_, exists := actualHeaders[cid]
			assert.True(t, exists, "cid: %s", cid)
		}
	})

	t.Run("block_parents", func(t *testing.T) {
		// expected count
		assert.Equal(t, len(expectedCIDs), len(strg.Data["block_parents"]))

		// models exist
		actualParents := make(map[string]struct{})
		for _, bh := range strg.Data["block_parents"] {
			actualParents[bh.(*blocks.BlockParent).Block] = struct{}{}
		}
		for _, cid := range expectedCIDs {
			_, exists := actualParents[cid]
			assert.True(t, exists, "cid: %s", cid)
		}
	})

	t.Run("drand_block_entries", func(t *testing.T) {
		// expected count
		assert.Equal(t, len(expectedRounds), len(strg.Data["drand_block_entries"]))

		// models exist
		actualParents := make(map[uint64]struct{})
		for _, bh := range strg.Data["drand_block_entries"] {
			actualParents[bh.(*blocks.DrandBlockEntrie).Round] = struct{}{}
		}
		for _, round := range expectedRounds {
			_, exists := actualParents[round]
			assert.True(t, exists, "round: %d", round)
		}
	})
}
