package raw

import (
	"context"
	"encoding/json"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/model"
	commonmodel "github.com/filecoin-project/lily/model/actors/common"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"
)

type rawActorStateChange struct {
	Address     address.Address
	StateChange *rawdiff.ActorChange
}

func TransformActorStates(ctx context.Context, s store.Store, current, executed *types.TipSet, actorMapRoot *cid.Cid) (model.Persistable, error) {
	if actorMapRoot == nil {
		return model.PersistableList{
			data.StartProcessingReport(tasktype.Actor, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
			data.StartProcessingReport(tasktype.ActorState, current).
				WithStatus(visormodel.ProcessingStatusInfo).
				WithInformation("no change detected").
				Finish(),
		}, nil
	}
	var out = model.PersistableList{}
	actorMap, err := adt.AsMap(s, *actorMapRoot)
	if err != nil {
		return nil, err
	}

	var actorChanges []*rawActorStateChange
	actorState := new(rawdiff.ActorChange)
	if err := actorMap.ForEach(actorState, func(key string) error {
		addr, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}
		val := new(rawdiff.ActorChange)
		*val = *actorState
		actorChanges = append(actorChanges, &rawActorStateChange{
			Address:     addr,
			StateChange: val,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	out = append(out, ActorStateHandler(ctx, current, executed, actorChanges))

	out = append(out, ActorHandler(ctx, current, executed, actorChanges))

	return out, nil
}

func ActorStateHandler(ctx context.Context, current, executed *types.TipSet, actors []*rawActorStateChange) model.Persistable {
	report := data.StartProcessingReport(tasktype.ActorState, current)
	// noop when actor is removed
	for _, actor := range actors {
		if actor.StateChange.Change == core.ChangeTypeRemove {
			continue
		}

		stateDump, err := vm.DumpActorState(util.ActorRegistry, actor.StateChange.Actor, actor.StateChange.Current)
		if err != nil {
			report.AddError(err)
			continue
		}

		state, err := json.Marshal(stateDump)
		if err != nil {
			report.AddError(err)
			continue
		}
		report.AddModels(&commonmodel.ActorState{
			Height: int64(current.Height()),
			Head:   actor.StateChange.Actor.Head.String(),
			Code:   actor.StateChange.Actor.Code.String(),
			State:  string(state),
		})
	}
	return report.Finish()
}

func ActorHandler(ctx context.Context, current, executed *types.TipSet, actors []*rawActorStateChange) model.Persistable {
	report := data.StartProcessingReport(tasktype.Actor, current)
	for _, actor := range actors {
		if actor.StateChange.Change == core.ChangeTypeRemove {
			continue
		}

		report.AddModels(&commonmodel.Actor{
			Height:    int64(current.Height()),
			ID:        actor.Address.String(),
			StateRoot: current.ParentState().String(),
			Code:      actor.StateChange.Actor.Code.String(),
			Head:      actor.StateChange.Actor.Head.String(),
			Balance:   actor.StateChange.Actor.Balance.String(),
			Nonce:     actor.StateChange.Actor.Nonce,
		})
	}
	return report.Finish()
}
