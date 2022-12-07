package procesor

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff"
	"github.com/filecoin-project/lily/pkg/extract/statetree"
	"github.com/filecoin-project/lily/tasks"
)

var (
	MinerCodes  = cid.NewSet()
	PowerCodes  = cid.NewSet()
	MarketCodes = cid.NewSet()
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
}

type ActorStateChanges struct {
	Current     *types.TipSet
	Executed    *types.TipSet
	Actors      map[address.Address]statetree.ActorDiff
	MinerActors map[address.Address]*minerdiff.StateDiff
}

func ProcessActorStateChanges(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*ActorStateChanges, error) {
	actorChanges, err := statetree.ActorChanges(ctx, api.Store(), current, executed)
	if err != nil {
		return nil, err
	}
	asc := &ActorStateChanges{
		Current:     current,
		Executed:    executed,
		Actors:      actorChanges,
		MinerActors: make(map[address.Address]*minerdiff.StateDiff, len(actorChanges)), // there are at most actorChanges entries
	}

	for addr, change := range actorChanges {
		if MinerCodes.Has(change.Current.Code) {
			minerChanges, err := minerdiff.State(ctx, api, &actors.ActorChange{
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
				return nil, err
			}
			asc.MinerActors[addr] = minerChanges
		}
	}
	return asc, nil

}
