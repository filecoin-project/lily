package extract

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gammazero/workerpool"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/datacapdiff"
	dcapDiffV1 "github.com/filecoin-project/lily/pkg/extract/actors/datacapdiff/v1"
	"github.com/filecoin-project/lily/pkg/extract/actors/initdiff"
	initDiffV1 "github.com/filecoin-project/lily/pkg/extract/actors/initdiff/v1"
	"github.com/filecoin-project/lily/pkg/extract/actors/marketdiff"
	marketDiffV1 "github.com/filecoin-project/lily/pkg/extract/actors/marketdiff/v1"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	minerDiffV1 "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v1"
	"github.com/filecoin-project/lily/pkg/extract/actors/powerdiff"
	powerDiffV1 "github.com/filecoin-project/lily/pkg/extract/actors/powerdiff/v1"
	"github.com/filecoin-project/lily/pkg/extract/actors/rawdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff"
	verifDiffV1 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v1"
	verifDiffV2 "github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff/v2"
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

func doWork(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, version actortypes.Version, results chan *StateDiffResult) {
	log.Infow("starting worker", "actor", act.Address)
	defer log.Infow("stopping worker", "actor", act.Address)
	select {
	case <-ctx.Done():
		log.Infow("canceling worker", "error", ctx.Err(), "actor", act.Address)
		return
	default:
		doRawActorDiff(ctx, api, act, version, results)
		//doActorDiff(ctx, api, act, version, results)
	}
}

func Actors(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, actorVersion actortypes.Version, workers int) (*ActorStateChanges, error) {
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

	pool := workerpool.New(workers)
	results := make(chan *StateDiffResult)
	wg := sync.WaitGroup{}
	workerCtx, cancel := context.WithCancel(ctx)
	activeWorkers := 0
	defer cancel()
	// sort actors on actor id in ascending order, causes market actor to be differed early, which is the slowest actor to diff.
	for _, addr := range sortedActorChangeKeys(actorChanges) {
		addr := addr
		change := actorChanges[addr]
		act := &actors.ActorChange{
			Address:  addr,
			Executed: change.Executed,
			Current:  change.Current,
			Type:     change.ChangeType,
		}
		wg.Add(1)
		activeWorkers++
		pool.Submit(func() {
			defer wg.Done()
			doWork(workerCtx, api, act, actorVersion, results)
			activeWorkers--
		})

	}
	go func() {
		log.Info("waiting for workers to complete")
		wg.Wait()
		log.Info("worker completed, closing worker channel")
		close(results)
	}()

	for stateDiff := range results {
		if err := stateDiff.Error; err != nil {
			log.Infow("activeworkers", "count", activeWorkers)
			log.Info("canceling workers")
			cancel()
			log.Info("canceled workers")
			// stop the pool
			log.Info("stopping worker pool")
			pool.StopWait()
			log.Info("stopped worker pool")
			log.Infow("activeworkers", "count", activeWorkers)
			<-results
			return nil, err
		}
		switch v := stateDiff.ActorDiff.(type) {
		case *rawdiff.StateDiffResult:
			asc.RawActors[stateDiff.Address] = v
		case *minerDiffV1.StateDiffResult:
			asc.MinerActors[stateDiff.Address] = v
		case *initDiffV1.StateDiffResult:
			asc.InitActor = v
		case *powerDiffV1.StateDiffResult:
			asc.PowerActor = v
		case *marketDiffV1.StateDiffResult:
			asc.MarketActor = v
		case *dcapDiffV1.StateDiffResult:
			asc.DatacapActor = v
		case *verifDiffV1.StateDiffResult, *verifDiffV2.StateDiffResult:
			asc.VerifregActor = v
		default:
			return nil, fmt.Errorf("unknown StateDiffResult type: %T", v)
		}
	}
	return asc, nil
}

func doRawActorDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, version actortypes.Version, results chan *StateDiffResult) {
	methods, handler, err := rawdiff.StateDiffFor(version)
	if err != nil {
		select {
		case <-ctx.Done():
			return
		case results <- &StateDiffResult{
			ActorDiff: nil,
			Address:   act.Address,
			Error:     err,
		}:
		}
		return
	}

	actorDiff := &actors.StateDiffer{
		Methods:      methods,
		ActorHandler: handler,
		ReportHandler: func(reports []actors.DifferReport) error {
			for _, report := range reports {
				log.Infow("reporting", "type", report.DiffType, "duration", report.Duration)
			}
			return nil
		},
	}
	res, err := actorDiff.ActorDiff(ctx, api, act)
	select {
	case <-ctx.Done():
		return
	case results <- &StateDiffResult{
		ActorDiff: res,
		Address:   act.Address,
		Error:     err,
	}:
	}
	return
}

func doActorDiff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange, version actortypes.Version, results chan *StateDiffResult) {
	var (
		methods []actors.ActorDiffMethods
		handler actors.ActorHandlerFn
		err     error
	)
	if core.DataCapCodes.Has(act.Current.Code) {
		methods, handler, err = datacapdiff.StateDiffFor(version)
	} else if core.MinerCodes.Has(act.Current.Code) {
		methods, handler, err = minerdiff.StateDiffFor(version)
	} else if core.InitCodes.Has(act.Current.Code) {
		methods, handler, err = initdiff.StateDiffFor(version)
	} else if core.PowerCodes.Has(act.Current.Code) {
		methods, handler, err = powerdiff.StateDiffFor(version)
	} else if core.MarketCodes.Has(act.Current.Code) {
		methods, handler, err = marketdiff.StateDiffFor(version)
	} else if core.VerifregCodes.Has(act.Current.Code) {
		methods, handler, err = verifregdiff.StateDiffFor(version)
	} else {
		return
	}

	if err != nil {
		select {
		case <-ctx.Done():
			return
		default:
			results <- &StateDiffResult{
				ActorDiff: nil,
				Address:   act.Address,
				Error:     err,
			}
		}
	}

	actorDiff := &actors.StateDiffer{
		Methods:      methods,
		ActorHandler: handler,
		ReportHandler: func(reports []actors.DifferReport) error {
			for _, report := range reports {
				log.Infow("reporting", "type", report.DiffType, "duration", report.Duration)
			}
			return nil
		},
	}
	res, err := actorDiff.ActorDiff(ctx, api, act)
	select {
	case <-ctx.Done():
		return
	default:
		results <- &StateDiffResult{
			ActorDiff: res,
			Address:   act.Address,
			Error:     err,
		}
	}
}
