package lotus

import (
	"context"
	"os"
	"strconv"

	lru "github.com/hashicorp/golang-lru"
	cid "github.com/ipfs/go-cid"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	miner "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
)

const (
	EnvBlockCacheSize  = "VISOR_API_BLK_CACHE"
	EnvTipSetCacheSize = "VISOR_API_TS_CACHE"
	EnvObjCacheSize    = "VISOR_API_OBJ_CACHE"
)

var (
	BlkCacheSize int64
	TsCacheSize  int64
	ObjCacheSize int64
)

func init() {
	BlkCacheSize = 1_000
	TsCacheSize = 1_000
	ObjCacheSize = 1_000

	if blkCacheStr := os.Getenv(EnvBlockCacheSize); blkCacheStr != "" {
		blkSize, err := strconv.ParseInt(blkCacheStr, 10, 64)
		if err != nil {
			log.Errorw("setting api block cache size", "error", err)
		} else {
			BlkCacheSize = blkSize
		}
	}

	if tsCacheStr := os.Getenv(EnvTipSetCacheSize); tsCacheStr != "" {
		tsSize, err := strconv.ParseInt(tsCacheStr, 10, 64)
		if err != nil {
			log.Errorw("setting api tipset cache size", "error", err)
		} else {
			TsCacheSize = tsSize
		}
	}
	if objCacheStr := os.Getenv(EnvObjCacheSize); objCacheStr != "" {
		objSize, err := strconv.ParseInt(objCacheStr, 10, 64)
		if err != nil {
			log.Errorw("setting api obj cache size", "error", err)
		} else {
			ObjCacheSize = objSize
		}
	}
}

func NewAPIWrapper(node api.FullNode, store adt.Store) (*APIWrapper, error) {
	blkCache, err := lru.NewARC(int(BlkCacheSize))
	if err != nil {
		return nil, err
	}
	tsCache, err := lru.NewARC(int(TsCacheSize))
	if err != nil {
		return nil, err
	}
	objCache, err := lru.NewARC(int(ObjCacheSize))
	return &APIWrapper{
		FullNode: node,
		store:    store,

		blkCache: blkCache,
		tsCache:  tsCache,
		objCache: objCache,
	}, nil
}

var _ lens.API = &APIWrapper{}

type APIWrapper struct {
	api.FullNode
	store adt.Store

	blkCache *lru.ARCCache
	tsCache  *lru.ARCCache
	objCache *lru.ARCCache
}

func (aw *APIWrapper) Store() adt.Store {
	return aw.store
}

func (aw *APIWrapper) ChainGetBlock(ctx context.Context, msg cid.Cid) (*types.BlockHeader, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainGetBlock")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.CacheType, "block"), tag.Upsert(metrics.CacheOp, "miss"))
	defer func() {stats.Record(ctx, metrics.CacheOpCount.M(1))}()

	v, hit := aw.blkCache.Get(msg)
	if hit {
		ctx, _ = tag.New(ctx, tag.Upsert(metrics.CacheOp, "hit"))
		return v.(*types.BlockHeader), nil
	}
	blk, err := aw.FullNode.ChainGetBlock(ctx, msg)
	if err != nil {
		return nil, err
	}
	aw.blkCache.Add(msg, blk)
	return blk, nil
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

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.CacheType, "tipset"), tag.Upsert(metrics.CacheOp, "miss"))
	defer func() {stats.Record(ctx, metrics.CacheOpCount.M(1))}()

	v, hit := aw.tsCache.Get(tsk)
	if hit {
		ctx, _ = tag.New(ctx, tag.Upsert(metrics.CacheOp, "hit"))
		return v.(*types.TipSet), nil
	}
	ts, err := aw.FullNode.ChainGetTipSet(ctx, tsk)
	if err != nil {
		return nil, err
	}
	aw.tsCache.Add(tsk, ts)
	return ts, nil
}

func (aw *APIWrapper) ChainNotify(ctx context.Context) (<-chan []*api.HeadChange, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainNotify")
	defer span.End()
	return aw.FullNode.ChainNotify(ctx)
}

func (aw *APIWrapper) ChainReadObj(ctx context.Context, obj cid.Cid) ([]byte, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.ChainReadObj")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.CacheType, "object"), tag.Upsert(metrics.CacheOp, "miss"))
	defer func() {stats.Record(ctx, metrics.CacheOpCount.M(1))}()

	v, hit := aw.objCache.Get(obj)
	if hit {
		ctx, _ = tag.New(ctx, tag.Upsert(metrics.CacheOp, "hit"))
		return v.([]byte), nil
	}
	raw, err := aw.FullNode.ChainReadObj(ctx, obj)
	if err != nil {
		return nil, err
	}
	aw.objCache.Add(obj, raw)
	return raw, nil
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

func (aw *APIWrapper) StateMinerSectors(ctx context.Context, addr address.Address, filter *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateMinerSectors")
	defer span.End()
	return aw.FullNode.StateMinerSectors(ctx, addr, filter, tsk)
}

func (aw *APIWrapper) StateReadState(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*api.ActorState, error) {
	ctx, span := global.Tracer("").Start(ctx, "Lotus.StateReadState")
	defer span.End()
	return aw.FullNode.StateReadState(ctx, actor, tsk)
}

func (aw *APIWrapper) ComputeGasOutputs(gasUsed, gasLimit int64, baseFee, feeCap, gasPremium abi.TokenAmount) vm.GasOutputs {
	return vm.ComputeGasOutputs(gasUsed, gasLimit, baseFee, feeCap, gasPremium)
}
