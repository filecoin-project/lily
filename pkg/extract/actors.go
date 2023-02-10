package extract

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-address"
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
	RawActors     map[address.Address]actors.DiffResult
	MinerActors   map[address.Address]actors.DiffResult
	DatacapActor  actors.DiffResult
	InitActor     actors.DiffResult
	MarketActor   actors.DiffResult
	PowerActor    actors.DiffResult
	VerifregActor actors.DiffResult
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
	ActorDiff actors.DiffResult
	Actor     *actors.Change
}

func Actors(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, workers int) (*ActorStateChanges, error) {
	actorChanges, err := statetree.ActorChanges(ctx, api.Store(), current, executed)
	if err != nil {
		return nil, err
	}

	curNtwkVersion, err := api.NetworkVersion(ctx, current.Key())
	if err != nil {
		return nil, err
	}
	exeNtwkVersion, err := api.NetworkVersion(ctx, executed.Key())
	if err != nil {
		return nil, err
	}
	curActorVersion, err := actortypes.VersionForNetwork(network.Version(curNtwkVersion))
	if err != nil {
		return nil, err
	}
	exeActorVersion, err := actortypes.VersionForNetwork(network.Version(exeNtwkVersion))
	if err != nil {
		return nil, err
	}
	log.Infow("extract actor states",
		"current_network_version", curNtwkVersion,
		"current_actor_version", curActorVersion,
		"current_height", current.Height(),
		"executed_network_version", exeNtwkVersion,
		"executed_actor_version", exeActorVersion,
		"executed_height", executed.Height(),
		"workers", workers)

	var (
		// pool is the worker pool used to manage workers
		pool = workerpool.New(workers)
		// wg is the wait-group used to synchronize workers
		wg = sync.WaitGroup{}
		// resCh is the channel actor diff workers return results on
		resCh = make(chan *StateDiffResult, workers*2)
		// errCh is the channel actor diff workers return errors on
		errCh = make(chan error, workers*2)
		// done is the channel used for signaling workers have stopped execution
		done = make(chan struct{})
		// cancel is the channel used to cancel any active workers
		cancel = make(chan struct{})
		// scheduledWorkers is the number of workers scheduled for execution, its value decreases as workers complete
		scheduledWorkers = 0
		// workerCtx is the context used by the workers.
		// TODO a method deep in the call stack of this function is not respecting context cancellation
		workerCtx = context.TODO()
	)

	for addr, change := range actorChanges {
		act := &actors.Change{
			Address:    addr,
			Executed:   change.Executed,
			ExeVersion: exeActorVersion,
			Current:    change.Current,
			CurVersion: curActorVersion,
			Type:       change.ChangeType,
		}

		// submit two workers, one to extract raw actor states and one to diff individual actor states
		wg.Add(1)
		scheduledWorkers++
		pool.Submit(func() {
			defer func() {
				wg.Done()
				scheduledWorkers--

			}()
			res, err := diffRawActorState(workerCtx, api, act)
			if err != nil {
				// attempt to send the error or bail if canceled
				select {
				case <-cancel:
					return
				case errCh <- err:
					return
				}
			}
			// attempt to send the result or bail if canceled
			select {
			case <-cancel:
				return
			case resCh <- res:
				return
			}
		})

		wg.Add(1)
		scheduledWorkers++
		pool.Submit(func() {
			defer func() {
				wg.Done()
				scheduledWorkers--

			}()
			res, ok, err := diffTypedActorState(workerCtx, api, act)
			if err != nil {
				// attempt to send the error or bail if canceled
				select {
				case <-cancel:
					return
				case errCh <- err:
					return
				}
			}
			if ok {
				// attempt to send the result or bail if canceled
				select {
				case <-cancel:
					return
				case resCh <- res:
					return
				}
			}
			// Not all actors have their state diffed, for example account actors have no state to diff: their state is empty.
			log.Debugw("no actor diff for actor", "current_network", curNtwkVersion, "executed_network", exeNtwkVersion,
				"address", act.Address, "current_code", act.Current.Code, "current_version",
				act.CurVersion, "executed_code", act.Executed.Code, "executed_version", act.ExeVersion)

		})
	}

	// wait for workers to complete then signal work is done.
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	// stop the worker pool, dropping any scheduled workers, and complete any wait-groups for the case a workers were canceled.
	cleanup := func() {
		pool.Stop()
		for i := 0; i < scheduledWorkers; i++ {
			wg.Done()
		}
	}

	// result of diffing all actor states
	out := &ActorStateChanges{
		Current:     current,
		Executed:    executed,
		MinerActors: make(map[address.Address]actors.DiffResult, len(actorChanges)), // there are at most actorChanges entries
		RawActors:   make(map[address.Address]actors.DiffResult, len(actorChanges)), // there are at most actorChanges entries
	}
	for {
		log.Infof("diff jobs todo %d", scheduledWorkers)
		select {
		// canceling the context or receiving an error causes all workers to stop and drops any scheduled workers.
		case <-workerCtx.Done():
			log.Errorw("context canceled", "error", workerCtx.Err())
			cancel <- struct{}{}
			cleanup()
			<-done
			return nil, workerCtx.Err()
		case err := <-errCh:
			log.Infow("worker received error while processing", "error", err)
			cancel <- struct{}{}
			cleanup()
			<-done
			return nil, err

			// happy path, all workers completed and we can return with a result
		case <-done:
			log.Info("done processing")
			cleanup()
			return out, nil

			// result returned from a worker corresponding to the actor it diffed.
		case res := <-resCh:
			switch v := res.ActorDiff.(type) {
			case *rawdiff.StateDiffResult:
				out.RawActors[res.Actor.Address] = v
			case *minerDiffV1.StateDiffResult:
				out.MinerActors[res.Actor.Address] = v
			case *initDiffV1.StateDiffResult:
				out.InitActor = v
			case *powerDiffV1.StateDiffResult:
				out.PowerActor = v
			case *marketDiffV1.StateDiffResult:
				out.MarketActor = v
			case *dcapDiffV1.StateDiffResult:
				out.DatacapActor = v
			case *verifDiffV1.StateDiffResult, *verifDiffV2.StateDiffResult:
				out.VerifregActor = v
			default:
				// this indicates a developer error so it's okay to panic, the code is wrong and must be fixed.
				panic(fmt.Errorf("unknown StateDiffResult type: %T", v))
			}
		}
	}

}

// diffRawActorState is called on every actor with a state change and returns their state.
func diffRawActorState(ctx context.Context, api tasks.DataSource, act *actors.Change) (*StateDiffResult, error) {
	methods, handler, err := rawdiff.StateDiffFor(act.CurVersion)
	if err != nil {
		return nil, err
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
	if err != nil {
		return nil, err
	}
	return &StateDiffResult{
		ActorDiff: res,
		Actor:     act,
	}, nil
}

// diffTypedActorState is called on every actor with a state change and returns artifacts from diffing their internal state
// such as miner hamts, market amts, etc. It returns false if the actor `act` doesn't have a corresponding state diff implementation.
func diffTypedActorState(ctx context.Context, api tasks.DataSource, act *actors.Change) (*StateDiffResult, bool, error) {
	var (
		methods []actors.ActorDiffMethods
		handler actors.ActorHandlerFn
		err     error
	)
	if core.DataCapCodes.Has(act.Current.Code) {
		methods, handler, err = datacapdiff.StateDiffFor(act.CurVersion)
	} else if core.MinerCodes.Has(act.Current.Code) {
		methods, handler, err = minerdiff.StateDiffFor(act.CurVersion)
	} else if core.InitCodes.Has(act.Current.Code) {
		methods, handler, err = initdiff.StateDiffFor(act.CurVersion)
	} else if core.PowerCodes.Has(act.Current.Code) {
		methods, handler, err = powerdiff.StateDiffFor(act.CurVersion)
	} else if core.MarketCodes.Has(act.Current.Code) {
		methods, handler, err = marketdiff.StateDiffFor(act.CurVersion)
	} else if core.VerifregCodes.Has(act.Current.Code) {
		methods, handler, err = verifregdiff.StateDiffFor(act.CurVersion)
	} else {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, nil
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
	if err != nil {
		return nil, false, err
	}
	return &StateDiffResult{
		ActorDiff: res,
		Actor:     act,
	}, true, nil
}
