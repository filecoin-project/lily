package util

import (
	"context"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lily/tasks"
	builtin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"

	lotusapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/blockstore"
)

func GetBlockReward(ctx context.Context, ds tasks.DataSource, tsk types.TipSetKey) (big.Int, error) {
	ract, err := ds.Actor(ctx, reward.Address, tsk)
	if err != nil {
		return big.Zero(), err
	}

	api := lotusapi.FullNodeStruct{}
	tbsRew := newTieredBlockstore(&api)

	rst, err := reward.Load(adt.WrapStore(ctx, cbor.NewCborStore(tbsRew)), ract)
	if err != nil {
		return big.Zero(), err
	}

	epochReward, err := rst.ThisEpochReward()
	if err != nil {
		return big.Zero(), err
	}

	// Divide by expected leaders per epoch to get block reward
	blockReward := types.BigDiv(epochReward, types.NewInt(uint64(builtin.ExpectedLeadersPerEpoch)))

	return blockReward, nil
}

func newTieredBlockstore(api *lotusapi.FullNodeStruct) blockstore.Blockstore {
	return blockstore.NewTieredBstore(
		blockstore.NewAPIBlockstore(api),
		blockstore.NewMemory(),
	)
}
