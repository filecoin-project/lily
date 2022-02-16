package tasks

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/lens"
)

// ChangeType denotes type of state change
type ChangeType int

const (
	ChangeTypeUnknown ChangeType = iota
	ChangeTypeAdd
	ChangeTypeRemove
	ChangeTypeModify
)

type ActorStateChange struct {
	Actor      types.Actor
	ChangeType ChangeType
}

type ActorStateChangeDiff map[string]ActorStateChange

type DataSource interface {
	Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	ActorState(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.ActorState, error)
	CirculatingSupply(ctx context.Context, ts *types.TipSet) (api.CirculatingSupply, error)
	MinerPower(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.MinerPower, error)
	ActorStateChanges(ctx context.Context, ts, pts *types.TipSet) (ActorStateChangeDiff, error)
	MessageExecutions(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error)
	ExecutedAndBlockMessages(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error)
	TipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error)
	Store() adt.Store
}
