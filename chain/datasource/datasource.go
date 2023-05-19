package datasource

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin"
	adt2 "github.com/filecoin-project/go-state-types/builtin/v10/util/adt"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"github.com/filecoin-project/lotus/chain/vm"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/singleflight"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/tasks"
)

var (
	tipsetMessageReceiptCacheSize int
	executedTsCacheSize           int
	diffPreCommitCacheSize        int
	diffSectorCacheSize           int
	actorCacheSize                int

	tipsetMessageReceiptSizeEnv = "LILY_TIPSET_MSG_RECEIPT_CACHE_SIZE"
	executedTsCacheSizeEnv      = "LILY_EXECUTED_TS_CACHE_SIZE"
	diffPreCommitCacheSizeEnv   = "LILY_DIFF_PRECOMMIT_CACHE_SIZE"
	diffSectorCacheSizeEnv      = "LILY_DIFF_SECTORS_CACHE_SIZE"
	actorCacheSizeEnv           = "LILY_ACTOR_CACHE_SIZE"
)

func getCacheSizeFromEnv(env string, defaultValue int) int {
	if s := os.Getenv(env); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			return int(v)
		}
		log.Warnf("invalid value (%s) for %s defaulting to %d: %s", s, env, defaultValue, err)
	}
	return defaultValue
}

func init() {
	tipsetMessageReceiptCacheSize = getCacheSizeFromEnv(tipsetMessageReceiptSizeEnv, 4)
	executedTsCacheSize = getCacheSizeFromEnv(executedTsCacheSizeEnv, 4)
	diffPreCommitCacheSize = getCacheSizeFromEnv(diffPreCommitCacheSizeEnv, 500)
	diffSectorCacheSize = getCacheSizeFromEnv(diffSectorCacheSizeEnv, 500)
	actorCacheSize = getCacheSizeFromEnv(actorCacheSizeEnv, 1000)
}

var _ tasks.DataSource = (*DataSource)(nil)

var log = logging.Logger("lily/datasource")

func NewDataSource(node lens.API) (*DataSource, error) {
	t := &DataSource{
		node: node,
	}
	var err error
	t.tsBlkMsgRecCache, err = lru.New(tipsetMessageReceiptCacheSize)
	if err != nil {
		return nil, err
	}

	t.executedTsCache, err = lru.New(executedTsCacheSize)
	if err != nil {
		return nil, err
	}

	// TODO these cache sizes will need to increase depending on the number of miner actors at each epoch
	t.diffPreCommitCache, err = lru.New(diffPreCommitCacheSize)
	if err != nil {
		return nil, err
	}

	t.diffSectorsCache, err = lru.New(diffSectorCacheSize)
	if err != nil {
		return nil, err
	}

	t.actorCache, err = lru.New(actorCacheSize)
	if err != nil {
		return nil, err
	}

	return t, nil
}

type DataSource struct {
	node lens.API

	executedTsCache *lru.Cache
	executedTsGroup singleflight.Group

	tsBlkMsgRecCache *lru.Cache
	tsBlkMsgRecGroup singleflight.Group

	diffSectorsCache *lru.Cache
	diffSectorsGroup singleflight.Group

	diffPreCommitCache *lru.Cache
	diffPreCommitGroup singleflight.Group

	actorCache *lru.Cache
}

func (t *DataSource) MessageReceiptEvents(ctx context.Context, root cid.Cid) ([]types.Event, error) {
	return t.node.ChainGetEvents(ctx, root)
}

func (t *DataSource) ComputeBaseFee(ctx context.Context, ts *types.TipSet) (abi.TokenAmount, error) {
	return t.node.ComputeBaseFee(ctx, ts)
}

func (t *DataSource) TipSetBlockMessages(ctx context.Context, ts *types.TipSet) ([]*lens.BlockMessages, error) {
	return t.node.MessagesForTipSetBlocks(ctx, ts)
}

func (t *DataSource) ChainGetMessagesInTipset(ctx context.Context, tsk types.TipSetKey) ([]api.Message, error) {
	return t.node.ChainGetMessagesInTipset(ctx, tsk)
}

func (t *DataSource) EthGetBlockByHash(ctx context.Context, blkHash ethtypes.EthHash, fullTxInfo bool) (ethtypes.EthBlock, error) {
	return t.node.EthGetBlockByHash(ctx, blkHash, fullTxInfo)
}

func (t *DataSource) EthGetTransactionReceipt(ctx context.Context, txHash ethtypes.EthHash) (*api.EthTxReceipt, error) {
	return t.node.EthGetTransactionReceipt(ctx, txHash)
}

// TipSetMessageReceipts returns the blocks and messages in `pts` and their corresponding receipts from `ts` matching block order in tipset (`pts`).
// TODO replace with lotus chainstore method when https://github.com/filecoin-project/lotus/pull/9186 lands
func (t *DataSource) TipSetMessageReceipts(ctx context.Context, ts, pts *types.TipSet) ([]*lens.BlockMessageReceipts, error) {
	key, err := asKey(ts, pts)
	if err != nil {
		return nil, err
	}
	value, found := t.tsBlkMsgRecCache.Get(key)
	if found {
		return value.([]*lens.BlockMessageReceipts), nil
	}

	value, err, _ = t.tsBlkMsgRecGroup.Do(key, func() (interface{}, error) {
		data, innerErr := t.node.TipSetMessageReceipts(ctx, ts, pts)
		if innerErr == nil {
			t.tsBlkMsgRecCache.Add(key, data)
		}
		return data, innerErr
	})
	if err != nil {
		return nil, err
	}

	return value.([]*lens.BlockMessageReceipts), nil
}

func (t *DataSource) TipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.TipSet")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", tsk.String()))
	}
	defer span.End()
	return t.node.ChainGetTipSet(ctx, tsk)
}

func (t *DataSource) Store() adt.Store {
	return t.node.Store()
}

func (t *DataSource) Actor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	metrics.RecordInc(ctx, metrics.DataSourceActorCacheRead)
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.Actor")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", tsk.String()))
		span.SetAttributes(attribute.String("address", addr.String()))
	}
	defer span.End()

	key, err := asKey(addr, tsk)
	if err != nil {
		return nil, err
	}
	value, found := t.actorCache.Get(key)
	if found {
		metrics.RecordInc(ctx, metrics.DataSourceActorCacheHit)
		return value.(*types.Actor), nil
	}

	act, err := t.node.StateGetActor(ctx, addr, tsk)
	if err == nil {
		t.actorCache.Add(key, act)
	}

	return act, err
}

func (t *DataSource) MinerPower(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.MinerPower, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.MinerPower")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.Key().String()))
		span.SetAttributes(attribute.String("address", addr.String()))
	}
	defer span.End()

	return t.node.StateMinerPower(ctx, addr, ts.Key())
}

func (t *DataSource) ActorState(ctx context.Context, addr address.Address, ts *types.TipSet) (*api.ActorState, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.ActorState")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.Key().String()))
		span.SetAttributes(attribute.String("address", addr.String()))
	}
	defer span.End()

	return t.node.StateReadState(ctx, addr, ts.Key())
}

func (t *DataSource) ActorStateChanges(ctx context.Context, ts, pts *types.TipSet) (tasks.ActorStateChangeDiff, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.ActorStateChanges")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.Key().String()))
		span.SetAttributes(attribute.String("parent", pts.Key().String()))
	}
	defer span.End()

	return GetActorStateChanges(ctx, t.Store(), ts, pts)
}

func (t *DataSource) CirculatingSupply(ctx context.Context, ts *types.TipSet) (api.CirculatingSupply, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.CirculatingSupply")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.Key().String()))
	}
	defer span.End()

	return t.node.CirculatingSupply(ctx, ts.Key())
}

func (t *DataSource) MessageExecutions(ctx context.Context, ts, pts *types.TipSet) ([]*lens.MessageExecution, error) {
	metrics.RecordInc(ctx, metrics.DataSourceMessageExecutionRead)
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.MessageExecutions")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.Key().String()))
		span.SetAttributes(attribute.String("parent", pts.Key().String()))
	}
	defer span.End()

	key, err := asKey(ts, pts)
	if err != nil {
		return nil, err
	}
	value, found := t.executedTsCache.Get(key)
	if found {
		metrics.RecordInc(ctx, metrics.DataSourceMessageExecutionCacheHit)
		return value.([]*lens.MessageExecution), nil
	}

	value, err, shared := t.executedTsGroup.Do(key, func() (interface{}, error) {
		data, innerErr := t.node.GetMessageExecutionsForTipSet(ctx, ts, pts)
		if innerErr == nil {
			t.executedTsCache.Add(key, data)
		}

		return data, innerErr
	})
	if span.IsRecording() {
		span.SetAttributes(attribute.Bool("shared", shared))
	}
	if err != nil {
		return nil, err
	}
	return value.([]*lens.MessageExecution), nil
}

func (t *DataSource) MinerLoad(store adt.Store, act *types.Actor) (miner.State, error) {
	return miner.Load(store, act)
}

func (t *DataSource) ShouldBurnFn(ctx context.Context, ts *types.TipSet) (lens.ShouldBurnFn, error) {
	return t.node.BurnFundsFn(ctx, ts)
}

func ComputeGasOutputs(ctx context.Context, block *types.BlockHeader, message *types.Message, receipt *types.MessageReceipt, shouldBurnFn lens.ShouldBurnFn) (vm.GasOutputs, error) {
	burn, err := shouldBurnFn(ctx, message, receipt.ExitCode)
	if err != nil {
		return vm.GasOutputs{}, err
	}
	return vm.ComputeGasOutputs(receipt.GasUsed, message.GasLimit, block.ParentBaseFee, message.GasFeeCap, message.GasPremium, burn), nil
}

type StateTreeMeta struct {
	// Root is the root of Map
	Root cid.Cid
	// Tree is the actual StateTree
	Tree *state.StateTree
}

func (s *StateTreeMeta) LoadMap(store adt.Store) (adt.Map, error) {
	return adt2.AsMap(store, s.Root, builtin.DefaultHamtBitwidth)
}

func LoadStateTreeMeta(ctx context.Context, s adt.Store, ts *types.TipSet) (*StateTreeMeta, error) {
	tree, err := state.LoadStateTree(s, ts.ParentState())
	if err != nil {
		return nil, err
	}

	root := getStateTreeRoot(ctx, s, ts)

	return &StateTreeMeta{
		Root: root,
		Tree: tree,
	}, nil

}

// getStateTreeRoot returns the cid of the state tree hamt. Required since the lotus types have unexported fields and we need to access the hamt root directly for fast diffing
func getStateTreeRoot(ctx context.Context, s adt.Store, ts *types.TipSet) cid.Cid {
	var root types.StateRoot
	// Try loading as a new-style state-tree (version/actors tuple).
	if err := s.Get(ctx, ts.ParentState(), &root); err != nil {
		// We failed to decode as the new version, must be an old version.
		return ts.ParentState()
	}
	return root.Actors
}

func GetActorStateChanges(ctx context.Context, store adt.Store, current, executed *types.TipSet) (tasks.ActorStateChangeDiff, error) {
	stop := metrics.Timer(ctx, metrics.DataSourceActorStateChangesDuration)
	defer stop()
	ctx, span := otel.Tracer("").Start(ctx, "GetActorStateChanges")
	defer span.End()

	start := time.Now()
	if executed.Height() == 0 {
		return GetGenesisActors(ctx, store, executed)
	}

	oldTree, err := LoadStateTreeMeta(ctx, store, executed)
	if err != nil {
		return nil, err
	}

	newTree, err := LoadStateTreeMeta(ctx, store, current)
	if err != nil {
		return nil, err
	}

	if newTree.Tree.Version() > 1 && oldTree.Tree.Version() > 1 {
		changes, err := fastDiff(ctx, store, oldTree, newTree)
		if err == nil {
			log.Infow("got actor state changes", "height", current.Height(), "duration", time.Since(start), "count", len(changes))
			return changes, nil
		}
	}
	log.Warnw("failed to diff state tree efficiently, falling back to slow method", "error", err)

	actors, err := state.Diff(ctx, oldTree.Tree, newTree.Tree)
	if err != nil {
		return nil, err
	}

	out := map[address.Address]tasks.ActorStateChange{}
	for addrStr, act := range actors {
		addr, err := address.NewFromString(addrStr)
		if err != nil {
			return nil, err
		}
		out[addr] = tasks.ActorStateChange{
			Actor:      act,
			ChangeType: tasks.ChangeTypeUnknown,
		}
	}
	log.Infow("got actor state changes", "height", current.Height(), "duration", time.Since(start), "count", len(out))
	return out, nil
}

func GetGenesisActors(ctx context.Context, store adt.Store, genesis *types.TipSet) (tasks.ActorStateChangeDiff, error) {
	out := map[address.Address]tasks.ActorStateChange{}
	tree, err := state.LoadStateTree(store, genesis.ParentState())
	if err != nil {
		return nil, err
	}
	if err := tree.ForEach(func(addr address.Address, act *types.Actor) error {
		out[addr] = tasks.ActorStateChange{
			Actor:      *act,
			ChangeType: tasks.ChangeTypeAdd,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func fastDiff(ctx context.Context, store adt.Store, oldTree, newTree *StateTreeMeta) (tasks.ActorStateChangeDiff, error) {
	oldMap, err := oldTree.LoadMap(store)
	if err != nil {
		return nil, err
	}
	newMap, err := newTree.LoadMap(store)
	if err != nil {
		return nil, err
	}
	changes, err := diff.Hamt(ctx, oldMap, newMap, store, store, hamt.UseTreeBitWidth(builtin.DefaultHamtBitwidth), hamt.UseHashFunction(func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}))
	if err != nil {
		return nil, err
	}

	buf := bytes.NewReader(nil)
	out := map[address.Address]tasks.ActorStateChange{}

	for _, change := range changes {
		addr, err := address.NewFromBytes([]byte(change.Key))
		if err != nil {
			return nil, fmt.Errorf("address in state tree was not valid: %w", err)
		}

		var ch tasks.ActorStateChange

		switch change.Type {
		case hamt.Add:
			ch.ChangeType = tasks.ChangeTypeAdd
			if newTree.Tree.Version() <= types.StateTreeVersion4 {
				var act types.ActorV4
				buf.Reset(change.After.Raw)
				err := act.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
				ch.Actor = *types.AsActorV5(&act)
			} else {
				var act types.Actor
				buf.Reset(change.After.Raw)
				err := act.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
				ch.Actor = act
			}

		case hamt.Remove:
			if newTree.Tree.Version() <= types.StateTreeVersion4 {
				var act types.ActorV4
				buf.Reset(change.Before.Raw)
				err := act.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
				ch.Actor = *types.AsActorV5(&act)
			} else {
				var act types.Actor
				buf.Reset(change.Before.Raw)
				err := act.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
				ch.Actor = act
			}

		case hamt.Modify:
			if newTree.Tree.Version() <= types.StateTreeVersion4 {
				var act types.ActorV4
				buf.Reset(change.After.Raw)
				err := act.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
				ch.Actor = *types.AsActorV5(&act)
			} else {
				var act types.Actor
				buf.Reset(change.After.Raw)
				err := act.UnmarshalCBOR(buf)
				buf.Reset(nil)
				if err != nil {
					return nil, err
				}
				ch.Actor = act
			}
		}
		out[addr] = ch
	}
	return out, nil
}

func asKey(strs ...fmt.Stringer) (string, error) {
	var sb strings.Builder
	for _, s := range strs {
		if _, err := sb.WriteString(s.String()); err != nil {
			return "", fmt.Errorf("failed to make key for %s: %w", s, err)
		}
	}
	return sb.String(), nil
}
