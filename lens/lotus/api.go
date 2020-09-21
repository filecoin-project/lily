package lotus

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	cid "github.com/ipfs/go-cid"
	"go.opentelemetry.io/otel/api/global"
)

func NewAPIWrapper(node api.FullNode, store adt.Store) *APIWrapper {
	return &APIWrapper{
		FullNode: node,
		store:    store,
	}
}

var _ lens.API = &APIWrapper{}

type APIWrapper struct {
	api.FullNode
	store adt.Store
}

func (aw *APIWrapper) Store() adt.Store {
	return aw.store
}

func (aw *APIWrapper) ChainGetBlock(ctx context.Context, msg cid.Cid) (*types.BlockHeader, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetBlock")
	defer span.End()
	return aw.FullNode.ChainGetBlock(ctx, msg)
}

func (aw *APIWrapper) ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetBlockMessages")
	defer span.End()
	return aw.FullNode.ChainGetBlockMessages(ctx, msg)
}

func (aw *APIWrapper) ChainGetGenesis(ctx context.Context) (*types.TipSet, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainNotify")
	defer span.End()
	return aw.FullNode.ChainGetGenesis(ctx)
}

func (aw *APIWrapper) ChainGetParentMessages(ctx context.Context, bcid cid.Cid) ([]api.Message, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetParentMessages")
	defer span.End()
	return aw.FullNode.ChainGetParentMessages(ctx, bcid)
}

func (aw *APIWrapper) ChainGetParentReceipts(ctx context.Context, bcid cid.Cid) ([]*types.MessageReceipt, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetParentReceipts")
	defer span.End()
	return aw.FullNode.ChainGetParentReceipts(ctx, bcid)
}

func (aw *APIWrapper) ChainGetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetTipSet")
	defer span.End()
	return aw.FullNode.ChainGetTipSet(ctx, tsk)
}

func (aw *APIWrapper) ChainNotify(ctx context.Context) (<-chan []*api.HeadChange, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainNotify")
	defer span.End()
	return aw.FullNode.ChainNotify(ctx)
}

func (aw *APIWrapper) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainReadObj")
	defer span.End()
	return aw.FullNode.ChainReadObj(ctx, obj)
}

func (aw *APIWrapper) StateChangedActors(ctx context.Context, old cid.Cid, new cid.Cid) (map[string]types.Actor, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateChangedActors")
	defer span.End()
	return aw.FullNode.StateChangedActors(ctx, old, new)
}

func (aw *APIWrapper) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateGetActor")
	defer span.End()
	return aw.FullNode.StateGetActor(ctx, actor, tsk)
}

func (aw *APIWrapper) StateListActors(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateListActors")
	defer span.End()
	return aw.FullNode.StateListActors(ctx, tsk)
}

func (aw *APIWrapper) StateMarketDeals(ctx context.Context, tsk types.TipSetKey) (map[string]api.MarketDeal, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateMarketDeals")
	defer span.End()
	return aw.FullNode.StateMarketDeals(ctx, tsk)
}

func (aw *APIWrapper) StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateMinerPower")
	defer span.End()
	return aw.FullNode.StateMinerPower(ctx, addr, tsk)
}

func (aw *APIWrapper) StateMinerSectors(ctx context.Context, addr address.Address, filter *bitfield.BitField, filterOut bool, tsk types.TipSetKey) ([]*api.ChainSectorInfo, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateMinerSectors")
	defer span.End()
	return aw.FullNode.StateMinerSectors(ctx, addr, filter, filterOut, tsk)
}

func (aw *APIWrapper) StateReadState(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*api.ActorState, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateReadState")
	defer span.End()
	return aw.FullNode.StateReadState(ctx, actor, tsk)
}
