package procesor

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/extract/actors/verifregdiff"
	"github.com/filecoin-project/lily/pkg/extract/statetree"
	"github.com/filecoin-project/lily/tasks"
)

var log = logging.Logger("lily/extract/processor")

var (
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
}

type ActorStateChanges struct {
	Current       *types.TipSet
	Executed      *types.TipSet
	Actors        map[address.Address]statetree.ActorDiff
	MinerActors   map[address.Address]*minerdiff.StateDiff
	VerifregActor map[address.Address]*verifregdiff.StateDiff
}

func (a ActorStateChanges) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int64("current", int64(a.Current.Height())),
		attribute.Int64("executed", int64(a.Executed.Height())),
		attribute.Int("actor_change", len(a.Actors)),
		attribute.Int("miner_changes", len(a.MinerActors)),
		attribute.Int("verifreg_changes", len(a.VerifregActor)),
	}
}

func (a ActorStateChanges) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range a.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

type StateDiffResult struct {
	ActorDiff actors.ActorStateDiff
	Address   address.Address
}

func ProcessActorStateChanges(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*ActorStateChanges, error) {
	actorChanges, err := statetree.ActorChanges(ctx, api.Store(), current, executed)
	if err != nil {
		return nil, err
	}
	asc := &ActorStateChanges{
		Current:       current,
		Executed:      executed,
		Actors:        actorChanges,
		MinerActors:   make(map[address.Address]*minerdiff.StateDiff, len(actorChanges)), // there are at most actorChanges entries
		VerifregActor: make(map[address.Address]*verifregdiff.StateDiff, len(actorChanges)),
	}

	grp, grpCtx := errgroup.WithContext(ctx)
	results := make(chan *StateDiffResult, len(actorChanges))
	for addr, change := range actorChanges {
		addr := addr
		change := change
		grp.Go(func() error {
			if MinerCodes.Has(change.Current.Code) {
				start := time.Now()
				minerChanges, err := minerdiff.State(grpCtx, api, &actors.ActorChange{
					Address:  addr,
					Executed: change.Executed,
					Current:  change.Current,
					Type:     change.ChangeType,
				},
					minerdiff.Debt{},
					minerdiff.Funds{},
					minerdiff.Info{},
					minerdiff.PreCommit{},
					minerdiff.Sectors{},
					minerdiff.SectorStatus{},
				)
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
				verifregChanges, err := verifregdiff.State(grpCtx, api, &actors.ActorChange{
					Address:  addr,
					Executed: change.Executed,
					Current:  change.Current,
					Type:     change.ChangeType,
				},
					// TODO the functions handed to these methods should be based on the epoch of the chain.
					//verifregdiff.Clients{},
					verifregdiff.Verifiers{},
					verifregdiff.Claims{},
				)
				if err != nil {
					return err
				}
				log.Infow("Extract VerifiedRegistry", "address", addr, "duration", time.Since(start))
				results <- &StateDiffResult{
					ActorDiff: verifregChanges,
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
		switch v := stateDiff.ActorDiff.(type) {
		case *minerdiff.StateDiff:
			asc.MinerActors[stateDiff.Address] = v
		case *verifregdiff.StateDiff:
			asc.VerifregActor[stateDiff.Address] = v
		}
	}
	return asc, nil

}
