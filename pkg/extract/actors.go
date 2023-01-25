package extract

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/datacapdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/initdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/marketdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/powerdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff"
	"github.com/filecoin-project/lily/pkg/extract/statetree"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/extract/processor")

type ActorStateChanges struct {
	Current       *types.TipSet
	Executed      *types.TipSet
	RawActors     map[address.Address]actors.ActorDiffResult
	MinerActors   map[address.Address]actors.ActorDiffResult
	DatacapActor  actors.ActorDiffResult
	InitActor     actors.ActorDiffResult
	MarketActor   actors.ActorDiffResult
	PowerActor    actors.ActorDiffResult
	VerifregActor actors.ActorDiffResult
}

func (a ActorStateChanges) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int64("current", int64(a.Current.Height())),
		attribute.Int64("executed", int64(a.Executed.Height())),
		attribute.Int("miner_changes", len(a.MinerActors)),
	}
}

func (a ActorStateChanges) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range a.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

type StateDiffResult struct {
	ActorDiff actors.ActorDiffResult
	Address   address.Address
	Error     error
}

type NetworkVersionGetter = func(ctx context.Context, epoch abi.ChainEpoch) network.Version

func DiffActor(ctx context.Context, api tasks.DataSource, version actortypes.Version, act *actors.ActorChange, diffLoader func(av actortypes.Version) (actors.ActorDiff, error)) (actors.ActorDiffResult, error) {
	// construct the state differ required by this actor version
	diff, err := diffLoader(version)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	// diff the actors state and collect changes
	log.Debugw("diff actor state", "address", act.Address.String(), "name", builtin.ActorNameByCode(act.Current.Code), "type", act.Type.String())
	diffRes, err := diff.State(ctx, api, act)
	if err != nil {
		return nil, err
	}
	log.Debugw("diffed actor state", "address", act.Address.String(), "name", builtin.ActorNameByCode(act.Current.Code), "type", act.Type.String(), "duration", time.Since(start))
	return diffRes, nil
}

func sortedActorChangeKeys(actors map[address.Address]statetree.ActorDiff) []address.Address {
	keys := make([]address.Address, 0, len(actors))

	for k := range actors {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		iKey, err := address.IDFromAddress(keys[i])
		if err != nil {
			panic(err)
		}
		jKey, err := address.IDFromAddress(keys[j])
		if err != nil {
			panic(err)
		}
		return iKey < jKey
	})

	return keys
}

func Actors(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, actorVersion actortypes.Version) (*ActorStateChanges, error) {
	actorChanges, err := statetree.ActorChanges(ctx, api.Store(), current, executed)
	if err != nil {
		return nil, err
	}
	asc := &ActorStateChanges{
		Current:     current,
		Executed:    executed,
		MinerActors: make(map[address.Address]actors.ActorDiffResult, len(actorChanges)), // there are at most actorChanges entries
		RawActors:   make(map[address.Address]actors.ActorDiffResult, len(actorChanges)), // there are at most actorChanges entries
	}

	pool := workerpool.New(8)
	results := make(chan *StateDiffResult, len(actorChanges))
	wg := sync.WaitGroup{}
	// sort actors on actor id in ascending order, causes market actor to be differed early, which is the slowest actor to diff.
	sortedKeys := sortedActorChangeKeys(actorChanges)
	for _, addr := range sortedKeys {
		addr := addr
		change := actorChanges[addr]
		act := &actors.ActorChange{
			Address:  addr,
			Executed: change.Executed,
			Current:  change.Current,
			Type:     change.ChangeType,
		}
		wg.Add(1)
		pool.Submit(func() {
			doActorDiff(ctx, api, act, actorVersion, results)
			wg.Done()
		})

	}
	go func() {
		wg.Wait()
		close(results)
	}()
	for stateDiff := range results {
		if stateDiff.Error != nil {
			pool.Stop()
			return nil, err
		}
		switch stateDiff.ActorDiff.Kind() {
		case "actor":
			asc.RawActors[stateDiff.Address] = stateDiff.ActorDiff
		case "miner":
			asc.MinerActors[stateDiff.Address] = stateDiff.ActorDiff
		case "verifreg":
			asc.VerifregActor = stateDiff.ActorDiff
		case "init":
			asc.InitActor = stateDiff.ActorDiff
		case "power":
			asc.PowerActor = stateDiff.ActorDiff
		case "market":
			asc.MarketActor = stateDiff.ActorDiff
		case "datacap":
			asc.DatacapActor = stateDiff.ActorDiff
		default:
			panic(stateDiff.ActorDiff.Kind())
		}
	}
	return asc, nil
}

func doActorDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, version actortypes.Version, results chan *StateDiffResult) {
	actorDiff := &rawdiff.StateDiff{
		DiffMethods: []actors.ActorStateDiff{rawdiff.Actor{}},
	}
	actorStateChanges, err := actorDiff.State(ctx, api, act)
	results <- &StateDiffResult{
		ActorDiff: actorStateChanges,
		Address:   act.Address,
		Error:     err,
	}
	var actorDiffer func(av actortypes.Version) (actors.ActorDiff, error)
	if core.DataCapCodes.Has(act.Current.Code) {
		actorDiffer = datacapdiff.StateDiffFor
	}
	if core.MinerCodes.Has(act.Current.Code) {
		actorDiffer = minerdiff.StateDiffFor
	}
	if core.VerifregCodes.Has(act.Current.Code) {
		actorDiffer = verifregdiff.StateDiffFor
	}
	if core.InitCodes.Has(act.Current.Code) {
		actorDiffer = initdiff.StateDiffFor
	}
	if core.PowerCodes.Has(act.Current.Code) {
		actorDiffer = powerdiff.StateDiffFor
	}
	if core.MarketCodes.Has(act.Current.Code) {
		actorDiffer = marketdiff.StateDiffFor
	}
	if actorDiffer == nil {
		return
	}

	res, err := DiffActor(ctx, api, version, act, actorDiffer)
	results <- &StateDiffResult{
		ActorDiff: res,
		Address:   act.Address,
		Error:     err,
	}
}
