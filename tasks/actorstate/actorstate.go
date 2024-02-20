package actorstate

import (
	"context"

	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/tasks"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

var log = logging.Logger("lily/tasks")

type ActorInfo struct {
	Actor      types.Actor
	ChangeType tasks.ChangeType
	Address    address.Address
	Current    *types.TipSet
	Executed   *types.TipSet
}

func (a ActorInfo) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("address", a.Address.String()),
		attribute.String("code", a.Actor.Code.String()),
		attribute.String("head", a.Actor.Head.String()),
		attribute.String("type", builtin.ActorNameByCode(a.Actor.Code)),
		attribute.String("change", a.ChangeType.String()),
	}
}

func (a ActorInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range a.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

// ActorStateAPI is the minimal subset of lens.API that is needed for actor state extraction
type ActorStateAPI interface {
	// TODO(optimize): StateGetActor is just a wrapper around StateManager.LoadActor with a lookup of the tipset which we already have
	Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)

	// TODO(optimize): StateMinerPower is just a wrapper for stmgr.GetPowerRaw which loads the power actor as we do in StoragePowerExtractor
	MinerPower(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.MinerPower, error)

	// TODO(optimize): StateReadState looks up the tipset and actor that we already have available
	ActorState(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.ActorState, error)

	// TODO(remove): StateMinerSectors loads the actor and then calls miner.Load which StorageMinerExtractor already has available
	// StateMinerSectors(ctx context.Context, addr address.Address, bf *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error)
	Store() adt.Store

	TipSetMessageReceipts(ctx context.Context, ts, pts *types.TipSet) ([]*lens.BlockMessageReceipts, error)

	DiffSectors(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.SectorChanges, error)
	DiffPreCommits(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChanges, error)
	DiffPreCommitsV8(ctx context.Context, addr address.Address, ts, pts *types.TipSet, pre, cur miner.State) (*miner.PreCommitChangesV8, error)

	MinerLoad(store adt.Store, act *types.Actor) (miner.State, error)

	LookupRobustAddress(ctx context.Context, idAddr address.Address, tsk types.TipSetKey) (address.Address, error)
}

// An ActorStateExtractor extracts actor state into a persistable format
type ActorStateExtractor interface {
	Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error)
}
