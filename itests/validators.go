package itests

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain"
	"github.com/filecoin-project/lily/lens/lily"
	"github.com/filecoin-project/lily/model/actors/common"
	"github.com/filecoin-project/lily/model/blocks"
	chain2 "github.com/filecoin-project/lily/model/chain"
	"github.com/filecoin-project/lily/model/messages"
	"github.com/filecoin-project/lily/storage"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var TaskModels = map[string][]string{
	chain.MessagesTask:       {"messages", "parsed_messages", "block_messages", "derived_gas_outputs", "message_gas_economy", "receipts"},
	chain.BlocksTask:         {"block_headers", "block_parents", "drand_block_entries"},
	chain.ChainConsensusTask: {"chain_consensus"},
	chain.ActorStatesRawTask: {"actors", "actor_states"},
}

var TaskValidators = map[string][]interface{}{
	chain.MessagesTask:       {BlockMessagesValidator{}, ReceiptsValidator{}},
	chain.BlocksTask:         {BlockHeaderValidator{}, BlockParentsValidator{}, DrandBlockEntriesValidator{}},
	chain.ChainConsensusTask: {ChainConsensusValidator{}},
	chain.ActorStatesRawTask: {ActorValidator{}, ActorStatesValidator{}},
}

type TipSetStateValidator interface {
	Validate(t *testing.T, state *TipSetState, strg *storage.Database)
}

type EpochValidator interface {
	Validate(t *testing.T, epoch int64, ts *types.TipSet, strg *storage.Database, api *lily.LilyNodeAPI)
}

var _ EpochValidator = (*ChainConsensusValidator)(nil)

type ChainConsensusValidator struct{}

func (ChainConsensusValidator) Validate(t *testing.T, epoch int64, ts *types.TipSet, strg *storage.Database, api *lily.LilyNodeAPI) {
	t.Run(fmt.Sprintf("chain_consensus_%d", epoch), func(t *testing.T) {
		ctx := context.Background()
		var m *chain2.ChainConsensus

		// Null round
		if ts == nil {
			// expect the parenttipset state at this epoch to be the last null round tipset and the tipset for the null
			// round to be null
			pts, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(epoch), types.EmptyTSK)
			require.NoError(t, err)

			exists, err := strg.AsORM().Model(m).
				Where("height = ?", epoch).
				Where("parent_state_root = ?", pts.ParentState().String()).
				Where("parent_tip_set = ?", pts.Parents().String()).
				Where("tip_set is null").
				Exists()
			require.NoError(t, err)
			assert.True(t, exists, "expected model with height %d, state_root %s", epoch, pts.ParentState())
		} else {
			exists, err := strg.AsORM().Model(m).
				Where("height = ?", epoch).
				Where("parent_state_root = ?", ts.ParentState().String()).
				Where("parent_tip_set = ?", ts.Parents().String()).
				Where("tip_set = ?", ts.Key().String()).
				Exists()
			require.NoError(t, err)
			assert.True(t, exists, "expected model with height %d, state_root %s", ts.Height(), ts.ParentState())

		}
		var count int
		_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM chain_consensus WHERE height = ?`, epoch)
		require.NoError(t, err)
		assert.EqualValues(t, 1, count)
	})
}

var _ TipSetStateValidator = (*DrandBlockEntriesValidator)(nil)

type DrandBlockEntriesValidator struct{}

func (DrandBlockEntriesValidator) Validate(t *testing.T, state *TipSetState, strg *storage.Database) {
	t.Run(fmt.Sprintf("drand_block_entries_%d", state.ts.Height()), func(t *testing.T) {
		for _, bh := range state.ts.Blocks() {
			for _, round := range bh.BeaconEntries {
				var m *blocks.DrandBlockEntrie
				exists, err := strg.AsORM().Model(m).
					Where("block = ?", bh.Cid().String()).
					Where("round = ?", round.Round).
					Exists()
				require.NoError(t, err)
				assert.True(t, exists, "expected model with cid: %s round: %d", bh.Cid(), round.Round)

				var count int
				_, err = strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM drand_block_entries WHERE block = ?`, bh.Cid().String())
				require.NoError(t, err)
				assert.Equal(t, len(bh.BeaconEntries), count)
			}

		}
	})
}

var _ TipSetStateValidator = (*ActorValidator)(nil)

type ActorValidator struct{}

func (ActorValidator) Validate(t *testing.T, state *TipSetState, strg *storage.Database) {
	t.Run(fmt.Sprintf("actor_%d", state.ts.Height()), func(t *testing.T) {
		for addr, act := range state.actorsChanges {
			var m *common.Actor
			exists, err := strg.AsORM().Model(m).
				Where("height = ?", state.ts.Height()).
				Where("state_root = ?", state.ts.ParentState().String()).
				Where("head = ?", act.Head.String()).
				Where("nonce = ?", act.Nonce).
				Where("balance = ?", act.Balance.String()).
				Where("id = ?", addr.String()).
				Exists()
			require.NoError(t, err)
			assert.Truef(t, exists, "expected model with height %d, state_root %s, head %s, address %s", state.ts.Height(), state.ts.ParentState(), act.Head, addr)
		}

		// the total number of actor models at this height should be equal to the number of actors in TipSetState.
		var count int
		_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM actors WHERE height = ?`, state.ts.Height())
		require.NoError(t, err)
		assert.Equal(t, len(state.actorsChanges), count)
	})
}

var _ TipSetStateValidator = (*ActorStatesValidator)(nil)

type ActorStatesValidator struct{}

func (ActorStatesValidator) Validate(t *testing.T, state *TipSetState, strg *storage.Database) {
	t.Run(fmt.Sprintf("actor_states_%d", state.ts.Height()), func(t *testing.T) {
		for _, act := range state.actorsChanges {
			var m *common.ActorState
			exists, err := strg.AsORM().Model(m).
				Where("height = ?", state.ts.Height()).
				Where("head = ?", act.Head.String()).
				Where("code = ?", act.Code.String()).
				Exists()
			require.NoError(t, err)
			assert.Truef(t, exists, "expected model with height %d, head %s, code %s", state.ts.Height(), act.Head, act.Code)
		}

		var count int
		_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM actor_states WHERE height = ?`, state.ts.Height())
		require.NoError(t, err)
		assert.Equal(t, len(state.actorsChanges), count)
	})
}

var _ TipSetStateValidator = (*BlockHeaderValidator)(nil)

type BlockHeaderValidator struct{}

func (BlockHeaderValidator) Validate(t *testing.T, state *TipSetState, strg *storage.Database) {
	t.Run(fmt.Sprintf("block_headers_%d", state.ts.Height()), func(t *testing.T) {
		for _, bh := range state.ts.Blocks() {
			var m *blocks.BlockHeader
			exists, err := strg.AsORM().Model(m).
				Where("cid = ?", bh.Cid().String()).
				Where("height = ?", int64(bh.Height)).
				Exists()
			require.NoError(t, err)
			assert.True(t, exists, "expected model with cid: %s height: %d", bh.Cid(), bh.Height)
		}

		var count int
		_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_headers WHERE height = ?`, state.ts.Height())
		require.NoError(t, err)
		assert.Equal(t, len(state.ts.Blocks()), count)
	})
}

var _ TipSetStateValidator = (*BlockMessagesValidator)(nil)

type BlockMessagesValidator struct{}

func (BlockMessagesValidator) Validate(t *testing.T, state *TipSetState, strg *storage.Database) {
	t.Run(fmt.Sprintf("block_messages_%d", state.ts.Height()), func(t *testing.T) {
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
		_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_messages WHERE height = ?`, state.ts.Height())
		require.NoError(t, err)
		assert.Equal(t, msgCount, count, "expect model with height = %d", state.ts.Height())
	})
}

var _ TipSetStateValidator = (*BlockParentsValidator)(nil)

type BlockParentsValidator struct{}

func (BlockParentsValidator) Validate(t *testing.T, state *TipSetState, strg *storage.Database) {
	t.Run(fmt.Sprintf("block_parents_%d", state.ts.Height()), func(t *testing.T) {
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
		_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM block_parents WHERE height = ?`, state.ts.Height())
		require.NoError(t, err)
		assert.Equal(t, totalBlockParents, count, "expect model with height = ?", state.ts.Height())
	})
}

var _ TipSetStateValidator = (*ReceiptsValidator)(nil)

type ReceiptsValidator struct{}

func (ReceiptsValidator) Validate(t *testing.T, state *TipSetState, strg *storage.Database) {
	t.Run(fmt.Sprintf("receipts_%d", state.ts.Height()), func(t *testing.T) {
		for msg, rect := range state.msgRects {
			var m *messages.Receipt
			exists, err := strg.AsORM().Model(m).
				Where("height = ?", state.ts.Height()).
				Where("message = ?", msg.String()).
				Where("state_root = ?", state.ts.ParentState().String()).
				Where("exit_code = ?", rect.ExitCode).
				Where("gas_used = ?", rect.GasUsed).
				Exists()
			require.NoError(t, err)
			assert.True(t, exists, "expected model with height %d, message %s", state.ts.Height(), msg.String())
		}

		var count int
		_, err := strg.AsORM().QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM receipts WHERE height = ?`, state.ts.Height())
		require.NoError(t, err)
		assert.Equal(t, len(state.msgRects), count)
	})
}
