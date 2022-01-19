//go:build calibnet
// +build calibnet

package calibnet

import (
	"context"
	"embed"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/chain"
	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/itests"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/lily"
	lutil "github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actors/common"
	"github.com/filecoin-project/lily/model/blocks"
	chain2 "github.com/filecoin-project/lily/model/chain"
	"github.com/filecoin-project/lily/model/messages"
	"github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/storage"
	tstorage "github.com/filecoin-project/lily/storage/testing"
	api2 "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

//go:embed vectors/*
var vectorData embed.FS // embeded test vector dir

type testCase struct {
	modelName string
	modeType  model.Persistable
	task      string
	test      func(ts *types.TipSet, state *TipSetState, strg *storage.Database, model interface{}, t *testing.T)
}

func TestLilyVectorWalkExtraction(t *testing.T) {
	// expect this test to finish under a min and at most 5 mins.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	lilyNode, strg, tipsetsWalked, cleanup := PerformTestVectorWalk(ctx, t, "vectors/calibnet_0-1000_full_state.car", vectorWalkTestCases...)
	defer cleanup()

	for _, ts := range tipsetsWalked {
		tsState, err := StateForTipSet(ctx, lilyNode, ts)
		require.NoError(t, err)

		for _, tc := range vectorWalkTestCases {
			t.Run(fmt.Sprintf("%s-height_%d", tc.modelName, ts.Height()), func(t *testing.T) {
				tc.test(ts, tsState, strg, tc.modeType, t)
			})
		}

		t.Run(fmt.Sprintf("processing_reports-height_%d", ts.Height()), func(t *testing.T) {
			/*
				if ts.Height() == 1000 {
					t.Skipf("Skipping height 1000 due to off by one-ness of tipset indexer.")
				}

			*/
			// validate the processing reports for all models walked.
			for _, tc := range vectorWalkTestCases {
				var m *visor.ProcessingReport
				exists, err := strg.AsORM().Model(m).
					Where("height = ?", ts.Height()).
					Where("task = ?", tc.task).
					Where("status = ?", visor.ProcessingStatusOK).
					Exists()
				require.NoError(t, err)
				assert.True(t, exists, "expected model with height %d, task %s, status %s", ts.Height(), tc.task, visor.ProcessingStatusOK)
				// TODO validate consensus tasks since there isn't a corresponding tipset height for null rounds.
				// TODO ensure there are no other models in the database. (e.g at a null round height or at a height greater max(ts.Height)
			}
		})
	}

}

var vectorWalkTestCases = []testCase{
	{
		modelName: "actors",
		modeType:  &common.Actor{},
		task:      chain.ActorStatesRawTask,
		test: func(ts *types.TipSet, state *TipSetState, strg *storage.Database, m interface{}, t *testing.T) {
			// ensure there exists a model for each actor listed in TipSetState.
			for addr, act := range state.actorsChanges {
				var m *common.Actor
				exists, err := strg.AsORM().Model(m).
					Where("height = ?", ts.Height()).
					Where("state_root = ?", ts.ParentState().String()).
					Where("head = ?", act.Head.String()).
					Where("nonce = ?", act.Nonce).
					Where("balance = ?", act.Balance.String()).
					Where("id = ?", addr.String()).
					Exists()
				require.NoError(t, err)
				assert.Truef(t, exists, "expected model with height %d, state_root %s, head %s, address %s", ts.Height(), ts.ParentState(), act.Head, addr)
			}

			// the total number of actor models at this height should be equal to the number of actors in TipSetState.
			var count int
			_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM actors WHERE height = ?`, ts.Height())
			require.NoError(t, err)
			assert.Equal(t, len(state.actorsChanges), count)
		},
	},
	{
		modelName: "actor_states",
		modeType:  &common.ActorState{},
		task:      chain.ActorStatesRawTask,
		test: func(ts *types.TipSet, state *TipSetState, strg *storage.Database, model interface{}, t *testing.T) {
			for _, act := range state.actorsChanges {
				var m *common.ActorState
				exists, err := strg.AsORM().Model(m).
					Where("height = ?", ts.Height()).
					Where("head = ?", act.Head.String()).
					Where("code = ?", act.Code.String()).
					Exists()
				require.NoError(t, err)
				assert.Truef(t, exists, "expected model with height %d, head %s, code %s", ts.Height(), act.Head, act.Code)
			}

			var count int
			_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM actor_states WHERE height = ?`, ts.Height())
			require.NoError(t, err)
			assert.Equal(t, len(state.actorsChanges), count)
		},
	},
	{
		modelName: "block_headers",
		modeType:  &blocks.BlockHeader{},
		task:      chain.BlocksTask,
		test: func(ts *types.TipSet, state *TipSetState, strg *storage.Database, model interface{}, t *testing.T) {
			for _, bh := range ts.Blocks() {
				var m *blocks.BlockHeader
				exists, err := strg.AsORM().Model(m).
					Where("cid = ?", bh.Cid().String()).
					Where("height = ?", int64(bh.Height)).
					Exists()
				require.NoError(t, err)
				assert.True(t, exists, "expected model with cid: %s height: %d", bh.Cid(), bh.Height)
			}

			var count int
			_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_headers WHERE height = ?`, ts.Height())
			require.NoError(t, err)
			assert.Equal(t, len(ts.Blocks()), count)
		},
	},
	{
		modelName: "block_messages",
		modeType:  &messages.BlockMessage{},
		task:      chain.MessagesTask,
		test: func(ts *types.TipSet, state *TipSetState, strg *storage.Database, model interface{}, t *testing.T) {
			var msgCount int
			for blk, msgs := range state.blockMsgs {
				msgCount += len(msgs.Cids)
				for _, msg := range msgs.Cids {
					var m *messages.BlockMessage
					exists, err := strg.AsORM().Model(m).
						Where("height = ?", blk.Height).
						Where("block = ?", blk.Cid().String()).
						Where("message = ?", msg.String()).
						Exists()
					require.NoError(t, err)
					assert.Truef(t, exists, "expected model with height %d, block %s, messages %s", blk.Height, blk.Cid(), msg)
				}
			}

			var count int
			_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_messages WHERE height = ?`, ts.Height())
			require.NoError(t, err)
			assert.Equal(t, msgCount, count, "expect model with height = %d", ts.Height())
		},
	},
	{
		modelName: "messages",
		modeType:  &messages.Messages{},
		task:      chain.MessagesTask,
		test: func(ts *types.TipSet, state *TipSetState, strg *storage.Database, model interface{}, t *testing.T) {
			var msgCount int
			for blk, msgs := range state.blockMsgs {
				msgCount += len(msgs.Cids)
				for _, msg := range msgs.Cids {
					var m *messages.Message
					exists, err := strg.AsORM().Model(m).
						Where("height = ?", blk.Height).
						Where("cid = ?", msg.String()).
						Exists()
					require.NoError(t, err)
					assert.Truef(t, exists, "expected model with height %d, block %s, messages %s", blk.Height, blk.Cid(), msg)
				}
			}

			var count int
			_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_messages WHERE height = ?`, ts.Height())
			require.NoError(t, err)
			assert.Equal(t, msgCount, count, "expect model with height = %d", ts.Height())
		},
	},
	{
		modelName: "block_parents",
		modeType:  &blocks.BlockParents{},
		task:      chain.BlocksTask,
		test: func(ts *types.TipSet, state *TipSetState, strg *storage.Database, model interface{}, t *testing.T) {
			totalBlockParents := 0
			for _, blk := range state.blocks {
				for _, parent := range blk.Parents {
					totalBlockParents++
					var m *blocks.BlockParent
					exists, err := strg.AsORM().Model(m).
						Where("height = ?", blk.Height).
						Where("block = ?", blk.Cid().String()).
						Where("parent = ?", parent.String()).
						Exists()
					require.NoError(t, err)
					assert.Truef(t, exists, "expected model with height %d, block %s, parent %s", blk.Height, blk.Cid(), parent)
				}
			}

			var count int
			_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_parents WHERE height = ?`, ts.Height())
			require.NoError(t, err)
			assert.Equal(t, totalBlockParents, count, "expect model with height = ?", ts.Height())

		},
	},
	{
		modelName: "chain_consensus",
		modeType:  &chain2.ChainConsensus{},
		task:      chain.ChainConsensusTask,
		test: func(ts *types.TipSet, state *TipSetState, strg *storage.Database, model interface{}, t *testing.T) {
			var m *chain2.ChainConsensus
			exists, err := strg.AsORM().Model(m).
				Where("height = ?", ts.Height()).
				Where("parent_state_root = ?", ts.ParentState().String()).
				Where("parent_tip_set = ?", ts.Parents().String()).
				Where("tip_set = ?", ts.Key().String()).
				Exists()
			require.NoError(t, err)
			assert.True(t, exists, "expected model with height %d, state_root %s", ts.Height(), ts.ParentState())

			var count int
			_, err = strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM chain_consensus WHERE height = ?`, ts.Height())
			require.NoError(t, err)
			assert.EqualValues(t, 1, count)
		},
	},
	{
		modelName: "receipts",
		modeType:  &messages.Receipts{},
		task:      chain.MessagesTask,
		test: func(ts *types.TipSet, state *TipSetState, strg *storage.Database, model interface{}, t *testing.T) {
			for msg, rect := range state.msgRects {
				var m *messages.Receipt
				exists, err := strg.AsORM().Model(m).
					Where("height = ?", ts.Height()).
					Where("message = ?", msg.String()).
					Where("state_root = ?", ts.ParentState().String()).
					Where("exit_code = ?", rect.ExitCode).
					Where("gas_used = ?", rect.GasUsed).
					Exists()
				require.NoError(t, err)
				assert.True(t, exists, "expected model with height %d, message %s", ts.Height(), msg.String())
			}

			var count int
			_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM receipts WHERE height = ?`, ts.Height())
			require.NoError(t, err)
			assert.Equal(t, len(state.msgRects), count)
		},
	},
}

// PerformTestVectorWalk constructs and instantiates a lily node with a datastore imported from the CAR file at `vectorPath`.
// Next it instructs the lily node to perform a walk for all tasks in `vectorTestCases` from its chain head to genesis.
// lastly it constructs a migrated storage and truncates all models provided in `vectorTestCases`.
// The lily node, storage, and a list of all tipset walked are returned.
func PerformTestVectorWalk(ctx context.Context, t testing.TB, vectorPath string, vectorTestCases ...testCase) (*lily.LilyNodeAPI, *storage.Database, []*types.TipSet, func()) {
	vectorFile, err := vectorData.Open(vectorPath)
	require.NoError(t, err)

	logging.SetAllLoggers(logging.LevelError)
	// when true all sql statements will be printed
	debugLogs := true
	if debugLogs {
		logging.SetAllLoggers(logging.LevelDebug)
	}

	// TODO the t.Cleanups can cause flaks, added a sleep but need a better solution here, maybe return a function that's call after all assertions are made?
	strg, strgCleanup := tstorage.WaitForExclusiveMigratedStorage(ctx, t, debugLogs)

	def := config.DefaultConf()
	ncfg := *def
	storageName := "TestDatabase1"
	ncfg.Storage = config.StorageConf{
		Postgresql: map[string]config.PgStorageConf{
			storageName: {
				URLEnv:          "LILY_TEST_DB",
				PoolSize:        20,
				ApplicationName: t.Name(),
				AllowUpsert:     false,
				SchemaName:      "public",
			},
		},
	}

	lilyAPI, apiCleanup := itests.NewTestNode(t, ctx, itests.TestNodeConfig{
		LilyConfig: &ncfg,
		CacheConfig: &lutil.CacheConfig{
			BlockstoreCacheSize: 0,
			StatestoreCacheSize: 0,
		},
		RepoPath:    t.TempDir(),
		Snapshot:    vectorFile,
		ApiEndpoint: "/ip4/127.0.0.1/tcp/4321",
	})

	api := lilyAPI.(*lily.LilyNodeAPI)

	head, err := api.ChainHead(ctx)
	require.NoError(t, err)

	// TODO fix this models thing when you refactor lily
	// since some models fall under same task name...
	tasks := make(map[string]struct{})
	models := []string{"visor_processing_reports"}
	for _, tc := range vectorTestCases {
		// processing reports doesn't have a task, ignore it since it always runs
		if tc.task == "" {
			continue
		}
		tasks[tc.task] = struct{}{}
		models = append(models, tc.modelName)
	}
	var walkTasks []string
	for task := range tasks {
		walkTasks = append(walkTasks, task)
	}

	truncateTable(t, strg.AsORM(), models...)

	walkCfg := &lily.LilyWalkConfig{
		From:                0,
		To:                  int64(head.Height()),
		Name:                t.Name(),
		Tasks:               walkTasks,
		Window:              0,
		RestartOnFailure:    false,
		RestartOnCompletion: false,
		RestartDelay:        0,
		Storage:             storageName,
	}

	res, err := api.LilyWalk(ctx, walkCfg)
	require.NoError(t, err)
	require.NotEmpty(t, res)

	// wait for the job to get to the scheduler else the job ID isn't found
	time.Sleep(3 * time.Second)
	ress, err := api.LilyJobWait(ctx, res.ID)
	require.NoError(t, err)
	require.NotEmpty(t, ress)

	tss, err := collectTipSets(ctx, api, head)
	require.NoError(t, err)

	return api, strg, tss, func() {
		if err := strgCleanup(); err != nil {
			t.Logf("failed to cleanup storage: %v", err)
		}
		if err := apiCleanup(ctx); err != nil {
			t.Logf("failed to cleanup api: %v", err)
		}
	}
}

// TipSetState contains the state of actors, blocks, messages, and receipts for the tipset it was derived from.
type TipSetState struct {
	// actors changed whiloe producing this tipset
	actorsChanges map[address.Address]types.Actor
	// blocks in this tipset
	blocks []*types.BlockHeader
	// messages in the blocks of this TipSet (will contain duplicate messages)
	blockMsgs map[*types.BlockHeader]*api2.BlockMessages
	// messages and their receipts
	msgRects map[cid.Cid]*types.MessageReceipt
}

// actorsFromGenesisBlock returns the set of actors found in the genesis block.
func actorsFromGenesisBlock(ctx context.Context, n *lily.LilyNodeAPI, ts *types.TipSet) (map[address.Address]types.Actor, error) {
	actors, err := n.StateListActors(ctx, ts.Key())
	if err != nil {
		return nil, err
	}

	actorsChanged := make(map[address.Address]types.Actor)
	for _, addr := range actors {
		act, err := n.StateGetActor(ctx, addr, ts.Key())
		if err != nil {
			return nil, err
		}
		actorsChanged[addr] = *act
	}
	return actorsChanged, nil
}

// StateForTipSet returns a TipSetState for TipSet `ts`. All state is derived from Lotus API calls.
func StateForTipSet(ctx context.Context, n *lily.LilyNodeAPI, ts *types.TipSet) (*TipSetState, error) {
	pts, err := n.ChainGetTipSet(context.TODO(), ts.Parents())
	if err != nil {
		return nil, err
	}

	actorsChanged := make(map[address.Address]types.Actor)
	if pts.Height() == 0 {
		actorsChanged, err = actorsFromGenesisBlock(ctx, n, ts)
		if err != nil {
			return nil, err
		}
	} else {
		// the actors who changed while producing this tipset
		tsActorChanges, err := n.StateChangedActors(ctx, pts.ParentState(), ts.ParentState())
		if err != nil {
			return nil, err
		}

		for addrStr, act := range tsActorChanges {
			addr, err := address.NewFromString(addrStr)
			if err != nil {
				return nil, err
			}
			actorsChanged[addr] = act
		}
	}

	// messages from the parent tipset, their receipts will be in (child) ts
	parentMessages, err := n.ChainAPI.Chain.MessagesForTipset(pts)
	if err != nil {
		return nil, err
	}

	blkMsgs := make(map[*types.BlockHeader]*api2.BlockMessages)
	msgRects := make(map[cid.Cid]*types.MessageReceipt)
	for _, blk := range ts.Blocks() {
		// map of blocks to their messages
		msgs, err := n.ChainGetBlockMessages(ctx, blk.Cid())
		if err != nil {
			return nil, err
		}
		blkMsgs[blk] = msgs

		// map of parent messages to their receipts
		for i := 0; i < len(parentMessages); i++ {
			r, err := n.ChainAPI.Chain.GetParentReceipt(blk, i)
			if err != nil {
				return nil, err
			}
			msgRects[parentMessages[i].Cid()] = r
		}

	}

	return &TipSetState{
		actorsChanges: actorsChanged,
		blocks:        ts.Blocks(),
		blockMsgs:     blkMsgs,
		msgRects:      msgRects,
	}, nil
}

// collectTipSets returns the list of ancestors of head including head.
func collectTipSets(ctx context.Context, api lens.API, head *types.TipSet) ([]*types.TipSet, error) {
	out := make([]*types.TipSet, 0, head.Height()+1)

	current := head
	for {
		parent, err := api.ChainGetTipSet(ctx, current.Parents())
		if err != nil {
			return nil, err
		}
		out = append(out, current)
		current = parent
		if current.Height() == 0 {
			break
		}
	}
	return out, nil
}

// truncateTables ensures the tables are truncated
func truncateTable(tb testing.TB, db *pg.DB, tableNames ...string) {
	for _, table := range tableNames {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
		require.NoError(tb, err, table)
	}
}
