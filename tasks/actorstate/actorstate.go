package actorstate

import (
	"context"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("actorstate")

type ActorInfo struct {
	Actor           types.Actor
	Address         address.Address
	ParentStateRoot cid.Cid
	Epoch           abi.ChainEpoch
	TipSet          types.TipSetKey
	ParentTipSet    types.TipSetKey
}

// ActorStateAPI is the minimal subset of lens.API that is needed for actor state extraction
type ActorStateAPI interface {
	ChainGetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error)
	ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error)
	StateGetReceipt(ctx context.Context, bcid cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error)
	ChainHasObj(ctx context.Context, obj cid.Cid) (bool, error)
	ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error)
	StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error)
	StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error)
	StateMinerSectors(ctx context.Context, addr address.Address, bf *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error)
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
	return builtin2.ActorNameByCode(code)
}
