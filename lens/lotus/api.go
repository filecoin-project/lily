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
	"github.com/opentracing/opentracing-go"
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.ChainGetBlock")
	defer span.Finish()
	return aw.FullNode.ChainGetBlock(ctx, msg)
}

func (aw *APIWrapper) ChainGetBlockMessages(ctx context.Context, msg cid.Cid) (*api.BlockMessages, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.ChainGetBlockMessages")
	defer span.Finish()
	return aw.FullNode.ChainGetBlockMessages(ctx, msg)
}

func (aw *APIWrapper) ChainGetGenesis(ctx context.Context) (*types.TipSet, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.ChainNotify")
	defer span.Finish()
	return aw.FullNode.ChainGetGenesis(ctx)
}

func (aw *APIWrapper) ChainGetParentMessages(ctx context.Context, bcid cid.Cid) ([]api.Message, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.ChainGetParentMessages")
	defer span.Finish()
	return aw.FullNode.ChainGetParentMessages(ctx, bcid)
}

func (aw *APIWrapper) ChainGetParentReceipts(ctx context.Context, bcid cid.Cid) ([]*types.MessageReceipt, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.ChainGetParentReceipts")
	defer span.Finish()
	return aw.FullNode.ChainGetParentReceipts(ctx, bcid)
}

func (aw *APIWrapper) ChainGetTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.ChainGetTipSet")
	defer span.Finish()
	return aw.FullNode.ChainGetTipSet(ctx, tsk)
}

func (aw *APIWrapper) ChainNotify(ctx context.Context) (<-chan []*api.HeadChange, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.ChainNotify")
	defer span.Finish()
	return aw.FullNode.ChainNotify(ctx)
}

func (aw *APIWrapper) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.ChainReadObj")
	defer span.Finish()
	return aw.FullNode.ChainReadObj(ctx, obj)
}

func (aw *APIWrapper) StateChangedActors(ctx context.Context, old cid.Cid, new cid.Cid) (map[string]types.Actor, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.StateChangedActors")
	defer span.Finish()
	return aw.FullNode.StateChangedActors(ctx, old, new)
}

func (aw *APIWrapper) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.StateGetActor")
	defer span.Finish()
	return aw.FullNode.StateGetActor(ctx, actor, tsk)
}

func (aw *APIWrapper) StateListActors(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.StateListActors")
	defer span.Finish()
	return aw.FullNode.StateListActors(ctx, tsk)
}

func (aw *APIWrapper) StateMarketDeals(ctx context.Context, tsk types.TipSetKey) (map[string]api.MarketDeal, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.StateMarketDeals")
	defer span.Finish()
	return aw.FullNode.StateMarketDeals(ctx, tsk)
}

func (aw *APIWrapper) StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.StateMinerPower")
	defer span.Finish()
	return aw.FullNode.StateMinerPower(ctx, addr, tsk)
}

func (aw *APIWrapper) StateMinerSectors(ctx context.Context, addr address.Address, filter *bitfield.BitField, filterOut bool, tsk types.TipSetKey) ([]*api.ChainSectorInfo, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Lotus.StateMinerSectors")
	defer span.Finish()
	return aw.FullNode.StateMinerSectors(ctx, addr, filter, filterOut, tsk)
}
