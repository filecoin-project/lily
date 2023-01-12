package procesor

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"

	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/actordiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/initdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/marketdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/powerdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff"
	"github.com/filecoin-project/lily/pkg/extract/statetree"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/extract/processor")

var (
	InitCodes     = cid.NewSet()
	MinerCodes    = cid.NewSet()
	PowerCodes    = cid.NewSet()
	MarketCodes   = cid.NewSet()
	VerifregCodes = cid.NewSet()
)

func init() {
	for _, c := range miner.AllCodes() {
		MinerCodes.Add(c)
	}
	for _, c := range power.AllCodes() {
		PowerCodes.Add(c)
	}
	for _, c := range market.AllCodes() {
		MarketCodes.Add(c)
	}
	for _, c := range verifreg.AllCodes() {
		VerifregCodes.Add(c)
	}
	for _, c := range init_.AllCodes() {
		InitCodes.Add(c)
	}
}

type ActorStateChanges struct {
	Current       *types.TipSet
	Executed      *types.TipSet
	ActorStates   map[address.Address]actors.ActorDiffResult
	MinerActors   map[address.Address]actors.ActorDiffResult
	VerifregActor actors.ActorDiffResult
	InitActor     actors.ActorDiffResult
	PowerActor    actors.ActorDiffResult
	MarketActor   actors.ActorDiffResult
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
}

type NetworkVersionGetter = func(ctx context.Context, epoch abi.ChainEpoch) network.Version

func ProcessActorStateChanges(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, nvg NetworkVersionGetter) (*ActorStateChanges, error) {
	actorChanges, err := statetree.ActorChanges(ctx, api.Store(), current, executed)
	if err != nil {
		return nil, err
	}
	asc := &ActorStateChanges{
		Current:     current,
		Executed:    executed,
		MinerActors: make(map[address.Address]actors.ActorDiffResult, len(actorChanges)), // there are at most actorChanges entries
		ActorStates: make(map[address.Address]actors.ActorDiffResult, len(actorChanges)), // there are at most actorChanges entries
	}

	actorVersion, err := core.ActorVersionForTipSet(ctx, current, nvg)
	if err != nil {
		return nil, err
	}

	grp, grpCtx := errgroup.WithContext(ctx)
	results := make(chan *StateDiffResult, len(actorChanges))
	for addr, change := range actorChanges {
		addr := addr
		change := change
		act := &actors.ActorChange{
			Address:  addr,
			Executed: change.Executed,
			Current:  change.Current,
			Type:     change.ChangeType,
		}
		grp.Go(func() error {
			actorDiff := &actordiff.StateDiff{
				DiffMethods: []actors.ActorStateDiff{actordiff.Actor{}},
			}
			actorStateChanges, err := actorDiff.State(grpCtx, api, act)
			if err != nil {
				return err
			}
			results <- &StateDiffResult{
				ActorDiff: actorStateChanges,
				Address:   addr,
			}
			if MinerCodes.Has(change.Current.Code) {
				start := time.Now()
				// construct the state differ required by this actor version
				actorDiff, err := minerdiff.StateDiffFor(actorVersion)
				if err != nil {
					return err
				}
				// diff the actors state and collect changes
				minerChanges, err := actorDiff.State(grpCtx, api, act)
				if err != nil {
					return err
				}
				log.Infow("Extract Miner", "address", addr, "duration", time.Since(start))
				results <- &StateDiffResult{
					ActorDiff: minerChanges,
					Address:   addr,
				}
			}
			if VerifregCodes.Has(change.Current.Code) {
				start := time.Now()
				// construct the state differ required by this actor version
				actorDiff, err := verifregdiff.StateDiffFor(actorVersion)
				if err != nil {
					return err
				}
				// diff the actors state and collect changes
				verifregChanges, err := actorDiff.State(grpCtx, api, act)
				if err != nil {
					return err
				}
				log.Infow("Extract VerifiedRegistry", "address", addr, "duration", time.Since(start))
				results <- &StateDiffResult{
					ActorDiff: verifregChanges,
					Address:   addr,
				}
			}
			if InitCodes.Has(change.Current.Code) {
				start := time.Now()
				actorDiff, err := initdiff.StateDiffFor(actorVersion)
				if err != nil {
					return err
				}
				initChanges, err := actorDiff.State(grpCtx, api, act)
				if err != nil {
					return err
				}
				log.Infow("Extracted Init", "address", addr, "duration", time.Since(start))
				results <- &StateDiffResult{
					ActorDiff: initChanges,
					Address:   addr,
				}
			}
			if PowerCodes.Has(change.Current.Code) {
				start := time.Now()
				actorDiff, err := powerdiff.StateDiffFor(actorVersion)
				if err != nil {
					return err
				}
				powerChanges, err := actorDiff.State(grpCtx, api, act)
				if err != nil {
					return err
				}
				log.Infow("Extracted Power", "address", addr, "duration", time.Since(start))
				results <- &StateDiffResult{
					ActorDiff: powerChanges,
					Address:   addr,
				}
			}
			if MarketCodes.Has(change.Current.Code) {
				start := time.Now()
				actorDiff, err := marketdiff.StateDiffFor(actorVersion)
				if err != nil {
					return err
				}
				marketChanges, err := actorDiff.State(grpCtx, api, act)
				if err != nil {
					return err
				}
				log.Infow("Extracted Market", "address", addr, "duration", time.Since(start))
				results <- &StateDiffResult{
					ActorDiff: marketChanges,
					Address:   addr,
				}
			}
			return nil
		})
	}
	go func() {
		if err := grp.Wait(); err != nil {
			log.Error(err)
		}
		close(results)
	}()
	for stateDiff := range results {
		switch stateDiff.ActorDiff.Kind() {
		case "actor":
			asc.ActorStates[stateDiff.Address] = stateDiff.ActorDiff
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
		default:
			panic(stateDiff.ActorDiff.Kind())
		}
	}
	return asc, nil

}
