package extract

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actortypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
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

func DiffActor(ctx context.Context, api tasks.DataSource, version actortypes.Version, act *actors.ActorChange, diffLoader func(av actortypes.Version) (actors.ActorDiff, error)) (*StateDiffResult, error) {
	// construct the state differ required by this actor version
	diff, err := diffLoader(version)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	// diff the actors state and collect changes
	diffRes, err := diff.State(ctx, api, act)
	if err != nil {
		return nil, err
	}
	log.Infow("diffed actor state", "address", act.Address.String(), "name", builtin.ActorNameByCode(act.Current.Code), "duration", time.Since(start))
	return &StateDiffResult{
		ActorDiff: diffRes,
		Address:   act.Address,
	}, nil
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
			actorDiff := &rawdiff.StateDiff{
				DiffMethods: []actors.ActorStateDiff{rawdiff.Actor{}},
			}
			actorStateChanges, err := actorDiff.State(grpCtx, api, act)
			if err != nil {
				return err
			}
			results <- &StateDiffResult{
				ActorDiff: actorStateChanges,
				Address:   act.Address,
			}
			if core.MinerCodes.Has(change.Current.Code) {
				res, err := DiffActor(ctx, api, actorVersion, act, minerdiff.StateDiffFor)
				if err != nil {
					return err
				}
				results <- res
			}
			if core.VerifregCodes.Has(change.Current.Code) {
				res, err := DiffActor(ctx, api, actorVersion, act, verifregdiff.StateDiffFor)
				if err != nil {
					return err
				}
				results <- res
			}
			if core.InitCodes.Has(change.Current.Code) {
				res, err := DiffActor(ctx, api, actorVersion, act, initdiff.StateDiffFor)
				if err != nil {
					return err
				}
				results <- res
			}
			if core.PowerCodes.Has(change.Current.Code) {
				res, err := DiffActor(ctx, api, actorVersion, act, powerdiff.StateDiffFor)
				if err != nil {
					return err
				}
				results <- res
			}
			if core.MarketCodes.Has(change.Current.Code) {
				res, err := DiffActor(ctx, api, actorVersion, act, marketdiff.StateDiffFor)
				if err != nil {
					return err
				}
				results <- res
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
		default:
			panic(stateDiff.ActorDiff.Kind())
		}
	}
	return asc, nil

}
