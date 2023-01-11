package v0

import (
	"context"
	"time"

	typegen "github.com/whyrusleeping/cbor-gen"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/pkg/extract/actors/generic"
	"github.com/filecoin-project/lily/tasks"
)

type DealChange struct {
	DealID   uint64
	Current  *typegen.Deferred
	Previous *typegen.Deferred
	Change   core.ChangeType
}

type DealChangeList []*DealChange

const KindMarketDeal = "market_deal"

func (p DealChangeList) Kind() actors.ActorStateKind {
	return KindMarketDeal
}

type Deals struct{}

func (Deals) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMarketDeal, zap.Inline(act), "duration", time.Since(start))
	}()
	return DiffDeals(ctx, api, act)
}

func DiffDeals(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	arrayChanges, err := generic.DiffActorArray(ctx, api, act, nil, nil)
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
