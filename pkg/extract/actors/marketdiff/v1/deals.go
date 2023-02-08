package v1

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-state-types/builtin/v10/util/adt"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

type DealChange struct {
	DealID   uint64            `cborgen:"dealID"`
	Current  *typegen.Deferred `cborgen:"current"`
	Previous *typegen.Deferred `cborgen:"previous"`
	Change   core.ChangeType   `cborgen:"change"`
}

type DealChangeList []*DealChange

const KindMarketDeal = "market_deal"

func (p DealChangeList) Kind() actors.ActorStateKind {
	return KindMarketDeal
}

func (p DealChangeList) ToAdtMap(store adt.Store, bw int) (cid.Cid, error) {
	node, err := adt.MakeEmptyMap(store, bw)
	if err != nil {
		return cid.Undef, err
	}
	for _, l := range p {
		if err := node.Put(abi.UIntKey(l.DealID), l); err != nil {
			return cid.Undef, err
		}
	}
	return node.Root()
}

type Deals struct{}

func (d Deals) Type() string {
	return KindMarketDeal
}

func (Deals) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMarketDeal, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffDeals(ctx, api, act)
}

func DiffDeals(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	arrayChanges, err := generic.DiffActorArray(ctx, api, act, MarketStateLoader, MarketDealsArrayLoader)
	if err != nil {
		return nil, err
	}
	out := make(DealChangeList, len(arrayChanges))
	for i, change := range arrayChanges {
		out[i] = &DealChange{
			DealID:   change.Key,
			Current:  change.Current,
			Previous: change.Previous,
			Change:   change.Type,
		}
	}
	return out, nil
}
