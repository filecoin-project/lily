package actorstate

import (
	"context"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/sentinel-visor/model"
)

var log = logging.Logger("actorstate")

type ActorInfo struct {
	Actor           types.Actor
	Address         address.Address
	ParentStateRoot cid.Cid
	Epoch           abi.ChainEpoch
	TipSet          *types.TipSet
	ParentTipSet    *types.TipSet
}

// ActorStateAPI is the minimal subset of lens.API that is needed for actor state extraction
type ActorStateAPI interface {
	ChainGetParentMessages(ctx context.Context, msg cid.Cid) ([]api.Message, error)
	StateGetReceipt(ctx context.Context, bcid cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error)

	// TODO(optimize): StateGetActor is just a wrapper around StateManager.LoadActor with a lookup of the tipset which we already have
	StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)

	// TODO(optimize): StateMinerPower is just a wrapper for stmgr.GetPowerRaw which loads the power actor as we do in StoragePowerExtractor
	StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error)

	// TODO(optimize): StateReadState looks up the tipset and actor that we already have available
	StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error)

	// TODO(remove): StateMinerSectors loads the actor and then calls miner.Load which StorageMinerExtractor already has available
	//StateMinerSectors(ctx context.Context, addr address.Address, bf *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error)
	Store() adt.Store
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

// ActorNameByCode returns the name of the actor code. Agnostic to the
// version of specs-actors.
func ActorNameByCode(code cid.Cid) string {
	if name := builtin.ActorNameByCode(code); name != "<unknown>" {
		return name
	}
	if name := builtin2.ActorNameByCode(code); name != "<unknown>" {
		return name
	}
	if name := builtin3.ActorNameByCode(code); name != "<unknown>" {
		return name
	}
	return builtin4.ActorNameByCode(code)
}
