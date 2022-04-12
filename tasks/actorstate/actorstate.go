package actorstate

import (
	"context"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/tasks")

type ActorInfo struct {
	Actor           types.Actor
	ChangeType      tasks.ChangeType
	Address         address.Address
	ParentStateRoot cid.Cid
	Epoch           abi.ChainEpoch
	TipSet          *types.TipSet
	ParentTipSet    *types.TipSet
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

	ExecutedAndBlockMessages(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error)
}

// An ActorStateExtractor extracts actor state into a persistable format
type ActorStateExtractor interface {
	Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error)
}

// All supported actor state extractors
var (
	extractorsMu sync.Mutex
	extractors   = map[cid.Cid]ActorStateExtractor{}
)

// Register adds an actor state extractor
func Register(code cid.Cid, e ActorStateExtractor) {
	extractorsMu.Lock()
	defer extractorsMu.Unlock()
	if _, ok := extractors[code]; ok {
		log.Warnf("extractor overrides previously registered extractor for code %q", code.String())
	}
	extractors[code] = e
}

func GetActorStateExtractor(code cid.Cid) (ActorStateExtractor, bool) {
	extractorsMu.Lock()
	defer extractorsMu.Unlock()
	ase, ok := extractors[code]
	return ase, ok
}
