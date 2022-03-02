package datasource

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"strconv"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/singleflight"
	"golang.org/x/xerrors"

	states0 "github.com/filecoin-project/specs-actors/actors/states"
	states2 "github.com/filecoin-project/specs-actors/v2/actors/states"
	states3 "github.com/filecoin-project/specs-actors/v3/actors/states"
	states4 "github.com/filecoin-project/specs-actors/v4/actors/states"
	states5 "github.com/filecoin-project/specs-actors/v5/actors/states"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/adt/diff"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/tasks"
)

var (
	executedBlkMsgCacheSize int
	executedTsCacheSize     int
	diffPreCommitCacheSize  int
	diffSectorCacheSize     int

	executedBlkMsgCacheSizeEnv = "LILY_EXECUTED_BLK_MSG_CACHE_SIZE"
	executedTsCacheSizeEnv     = "LILY_EXECUTED_TS_CACHE_SIZE"
	diffPreCommitCacheSizeEnv  = "LILY_DIFF_PRECOMMIT_CACHE_SIZE"
	diffSectorCacheSizeEnv     = "LILY_DIFF_SECTORS_CACHE_SIZE"
)

func init() {
	executedBlkMsgCacheSize = 4
	executedTsCacheSize = 4
	diffPreCommitCacheSize = 500
	diffSectorCacheSize = 500
	if s := os.Getenv(executedBlkMsgCacheSizeEnv); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			executedBlkMsgCacheSize = int(v)
		} else {
			log.Warnf("invalid value (%s) for %s defaulting to %d: %s", s, executedBlkMsgCacheSizeEnv, executedBlkMsgCacheSize, err)
		}
	}
	if s := os.Getenv(executedTsCacheSizeEnv); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			executedTsCacheSize = int(v)
		} else {
			log.Warnf("invalid value (%s) for %s defaulting to %d: %s", s, executedTsCacheSizeEnv, executedTsCacheSize, err)
		}
	}
	if s := os.Getenv(diffPreCommitCacheSizeEnv); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			diffPreCommitCacheSize = int(v)
		} else {
			log.Warnf("invalid value (%s) for %s defaulting to %d: %s", s, diffPreCommitCacheSizeEnv, diffPreCommitCacheSize, err)
		}
	}
	if s := os.Getenv(diffSectorCacheSizeEnv); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			diffSectorCacheSize = int(v)
		} else {
			log.Warnf("invalid value (%s) for %s defaulting to %d: %s", s, diffSectorCacheSizeEnv, diffSectorCacheSize, err)
		}
	}

}

var _ tasks.DataSource = (*DataSource)(nil)

var log = logging.Logger("lily/datasource")

type DataSource struct {
	node lens.API

	executedBlkMsgCache *lru.Cache
	executedBlkMsgGroup singleflight.Group

	executedTsCache *lru.Cache
	executedTsGroup singleflight.Group

	diffSectorsCache *lru.Cache
	diffSectorsGroup singleflight.Group

	diffPreCommitCache *lru.Cache
	diffPreCommitGroup singleflight.Group
}

func NewDataSource(node lens.API) (*DataSource, error) {
	t := &DataSource{
		node: node,
	}
	var err error
	t.executedBlkMsgCache, err = lru.New(executedBlkMsgCacheSize)
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

	return t, nil
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
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.Actor")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", tsk.String()))
		span.SetAttributes(attribute.String("address", addr.String()))
	}
	defer span.End()
	return t.node.StateGetActor(ctx, addr, tsk)
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

	key := ts.Key().String() + pts.Key().String()
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

func (t *DataSource) ExecutedAndBlockMessages(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error) {
	metrics.RecordInc(ctx, metrics.DataSourceExecutedAndBlockMessagesRead)
	ctx, span := otel.Tracer("").Start(ctx, "DataSource.ExecutedAndBlockMessages")
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tipset", ts.Key().String()))
		span.SetAttributes(attribute.String("parent", pts.Key().String()))
	}
	defer span.End()

	key := ts.Key().String() + pts.Key().String()
	value, found := t.executedBlkMsgCache.Get(key)
	if found {
		metrics.RecordInc(ctx, metrics.DataSourceExecutedAndBlockMessagesCacheHit)
		return value.(*lens.TipSetMessages), nil
	}

	value, err, shared := t.executedBlkMsgGroup.Do(key, func() (interface{}, error) {
		data, innerErr := t.node.GetExecutedAndBlockMessagesForTipset(ctx, ts, pts)
		if innerErr == nil {
			t.executedBlkMsgCache.Add(key, data)
		}

		return data, innerErr
	})

	if span.IsRecording() {
		span.SetAttributes(attribute.Bool("shared", shared))
	}
	if err != nil {
		return nil, err
	}
	return value.(*lens.TipSetMessages), nil
}

func GetActorStateChanges(ctx context.Context, store adt.Store, current, executed *types.TipSet) (tasks.ActorStateChangeDiff, error) {
	ctx, span := otel.Tracer("").Start(ctx, "GetActorStateChanges")
	defer span.End()

	start := time.Now()
	if executed.Height() == 0 {
		return GetGenesisActors(ctx, store, executed)
	}

	// we have this special method here to get the HAMT node root required by the faster diffing logic. I hate this.
	oldRoot, oldVersion, err := getStateTreeHamtRootCIDAndVersion(ctx, store, executed.ParentState())
	if err != nil {
		return nil, err
	}
	newRoot, newVersion, err := getStateTreeHamtRootCIDAndVersion(ctx, store, current.ParentState())
	if err != nil {
		return nil, err
	}

	if newVersion == oldVersion && (newVersion != types.StateTreeVersion0 && newVersion != types.StateTreeVersion1) {
		changes, err := fastDiff(ctx, store, oldRoot, newRoot)
		if err == nil {
			metrics.RecordInc(ctx, metrics.DataSourceActorStateChangesFastDiff)
			log.Infow("got actor state changes", "height", current.Height(), "duration", time.Since(start), "count", len(changes))
			if span.IsRecording() {
				span.SetAttributes(attribute.Bool("fast", true), attribute.Int("changes", len(changes)))
			}
			return changes, nil
		}
		log.Warnw("failed to diff state tree efficiently, falling back to slow method", "error", err)
	}
	metrics.RecordInc(ctx, metrics.DataSourceActorStateChangesSlowDiff)

	oldTree, err := state.LoadStateTree(store, executed.ParentState())
	if err != nil {
		return nil, err
	}

	newTree, err := state.LoadStateTree(store, current.ParentState())
	if err != nil {
		return nil, err
	}

	actors, err := state.Diff(ctx, oldTree, newTree)
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
	span.SetAttributes(attribute.Bool("fast", true), attribute.Int("changes", len(out)))
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

func fastDiff(ctx context.Context, store adt.Store, oldR, newR adt.Map) (tasks.ActorStateChangeDiff, error) {
	// TODO: replace hamt.UseTreeBitWidth and hamt.UseHashFunction with values based on network version
	changes, err := diff.Hamt(ctx, oldR, newR, store, store, hamt.UseTreeBitWidth(5), hamt.UseHashFunction(func(input []byte) []byte {
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
			return nil, xerrors.Errorf("address in state tree was not valid: %w", err)
		}
		var ch tasks.ActorStateChange
		switch change.Type {
		case hamt.Add:
			ch.ChangeType = tasks.ChangeTypeAdd
			buf.Reset(change.After.Raw)
			err = ch.Actor.UnmarshalCBOR(buf)
			buf.Reset(nil)
			if err != nil {
				return nil, err
			}
		case hamt.Remove:
			ch.ChangeType = tasks.ChangeTypeRemove
			buf.Reset(change.Before.Raw)
			err = ch.Actor.UnmarshalCBOR(buf)
			buf.Reset(nil)
			if err != nil {
				return nil, err
			}
		case hamt.Modify:
			ch.ChangeType = tasks.ChangeTypeModify
			buf.Reset(change.After.Raw)
			err = ch.Actor.UnmarshalCBOR(buf)
			buf.Reset(nil)
			if err != nil {
				return nil, err
			}
		}
		out[addr] = ch
	}
	return out, nil
}

func getStateTreeHamtRootCIDAndVersion(ctx context.Context, store adt.Store, c cid.Cid) (adt.Map, types.StateTreeVersion, error) {
	var root types.StateRoot
	// Try loading as a new-style state-tree (version/actors tuple).
	if err := store.Get(ctx, c, &root); err != nil {
		// We failed to decode as the new version, must be an old version.
		root.Actors = c
		root.Version = types.StateTreeVersion0
	}

	switch root.Version {
	case types.StateTreeVersion0:
		var tree *states0.Tree
		tree, err := states0.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	case types.StateTreeVersion1:
		var tree *states2.Tree
		tree, err := states2.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	case types.StateTreeVersion2:
		var tree *states3.Tree
		tree, err := states3.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	case types.StateTreeVersion3:
		var tree *states4.Tree
		tree, err := states4.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	case types.StateTreeVersion4:
		var tree *states5.Tree
		tree, err := states5.LoadTree(store, root.Actors)
		if err != nil {
			return nil, 0, err
		}
		return tree.Map, root.Version, nil
	default:
		return nil, 0, xerrors.Errorf("unsupported state tree version: %d", root.Version)
	}
}
