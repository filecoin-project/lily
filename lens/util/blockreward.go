package util

import (
	"context"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/tasks"
	builtin "github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	lotusapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/blockstore"
)

func GetBlockReward(ctx context.Context, ds tasks.DataSource, tsk types.TipSetKey, store adt.Store) (big.Int, error) {
	ract, err := ds.Actor(ctx, reward.Address, tsk)
	if err != nil {
		log.Errorf("[GetBlockReward] failed to get reward actor: %v", err)
		return big.Zero(), err
	}

	rewardState, err := reward.Load(store, ract)
	if err != nil {
		log.Errorf("[GetBlockReward] failed to load reward actor state: %v", err)
		return big.Zero(), err
	}

	epochReward, err := rewardState.ThisEpochReward()
	if err != nil {
		log.Errorf("[GetBlockReward] failed to get ThisEpochReward: %v", err)
		return big.Zero(), err
	}

	// Divide by expected leaders per epoch to get block reward
	blockReward := types.BigDiv(epochReward, types.NewInt(uint64(builtin.ExpectedLeadersPerEpoch)))

	log.Infof("[GetBlockReward] block reward: %s", blockReward.String())

	return blockReward, nil
}

func newTieredBlockstore(api *lotusapi.FullNodeStruct) blockstore.Blockstore {
	return blockstore.NewTieredBstore(
		blockstore.NewAPIBlockstore(api),
		blockstore.NewMemory(),
	)
}
