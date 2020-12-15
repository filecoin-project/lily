package actorstate_test

import (
	"context"
	"github.com/filecoin-project/lotus/chain/types"
	minermodel "github.com/filecoin-project/sentinel-visor/model/actors/miner"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMinerState(t *testing.T) {
	mapi := NewMockAPI(t)
	ctx := context.Background()
	minerAddr := tutils.NewIDAddr(t, 123)
	parentTipset := mapi.fakeTipset(minerAddr, 0)
	curTipset := mapi.fakeTipset(minerAddr, 1)

	// load the test vector
	minerVector := mapi.MinerVectorForHead("bafy2bzacea4cfici4adqn2qmtxy4kdmrmy5bbr4567yiies6ogpgvhnev6qb2", curTipset)

	// create an empty actor state for the vector.
	emptyState := mapi.mustCreateEmptyMinerStateV0()
	emptyHead, err := mapi.store.Put(ctx, emptyState)
	require.NoError(t, err)
	emptyInfo := &types.Actor{
		Code:    builtin.StorageMinerActorCodeID,
		Head:    emptyHead,
		Nonce:   0,
		Balance: types.NewInt(0),
	}
	mapi.setActor(parentTipset.Key(), minerVector.Info.Address, emptyInfo)
	mapi.setActor(curTipset.Key(), minerVector.Info.Address, minerVector.Info.Actor)

	// create an actorInfo to extract from using the vector state.
	actorInfo := actorstate.ActorInfo{
		Actor:           *minerVector.Info.Actor,
		Address:         minerVector.Info.Address,
		Epoch:           1,
		TipSet:          curTipset.Key(),
		ParentStateRoot: curTipset.ParentState(),
		ParentTipSet:    parentTipset.Key(),
	}

	// extract the actor state.
	ex := actorstate.StorageMinerExtractor{}
	model, err := ex.Extract(ctx, actorInfo, mapi)
	require.NoError(t, err)

	extractedMiner, ok := model.(*minermodel.MinerTaskResult)
	require.True(t, ok)

	_ = extractedMiner

}
