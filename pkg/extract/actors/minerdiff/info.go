package minerdiff

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/tasks"
)

type InfoChange struct {
	Info typegen.Deferred
}

func (i *InfoChange) Kind() ActorStateKind {
	return "miner_info"
}

type Info struct{}

func (Info) Diff(ctx context.Context, api tasks.DataSource, act *core.ActorChange, current, executed *types.TipSet) (ActorStateChange, error) {
	return InfoDiff(ctx, api, act, executed)
}

type DiffInfoAPI interface {
	Store() adt.Store
	ChainReadObj(ctx context.Context, c cid.Cid) ([]byte, error)
	MinerLoad(store adt.Store, act *types.Actor) (miner.State, error)
	Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
}

// separate method for testing purposes

func InfoDiff(ctx context.Context, api DiffInfoAPI, act *core.ActorChange, executed *types.TipSet) (*InfoChange, error) {
	// was removed, no new info
	if act.Type == core.ChangeTypeRemove {
		return nil, nil
	}
	currentMiner, err := api.MinerLoad(api.Store(), act.Actor)
	if err != nil {
		return nil, err
	}
	infoBytes, err := api.ChainReadObj(ctx, currentMiner.InfoCid())
	if err != nil {
		return nil, err
	}
	// was added, info is new
	if act.Type == core.ChangeTypeAdd {
		return &InfoChange{
			Info: typegen.Deferred{Raw: infoBytes},
		}, nil
	}
	// actor state was modified, check if miner info changed
	executedState, err := api.Actor(ctx, act.Address, executed.Key())
	if err != nil {
		return nil, err
	}
	executedMiner, err := api.MinerLoad(api.Store(), executedState)
	if err != nil {
		return nil, err
	}
	// wasn't modified
	if executedMiner.InfoCid().Equals(currentMiner.InfoCid()) {
		return nil, nil
	}
	return &InfoChange{
		Info: typegen.Deferred{Raw: infoBytes},
	}, nil
}
