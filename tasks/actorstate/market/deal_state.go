package market

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/model"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var _ actorstate.ActorStateExtractor = (*DealStateExtractor)(nil)

type DealStateExtractor struct{}

func (DealStateExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "DealStateExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "DealStateExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMarketStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	currDealStates, err := ec.CurrState.States()
	if err != nil {
		return nil, fmt.Errorf("loading current market deal states: %w", err)
	}

	if ec.IsGenesis() {
		var out marketmodel.MarketDealStates
		if err := currDealStates.ForEach(func(id abi.DealID, ds market.DealState) error {
			out = append(out, &marketmodel.MarketDealState{
				Height:           int64(ec.CurrTs.Height()),
				DealID:           uint64(id),
				SectorStartEpoch: int64(ds.SectorStartEpoch),
				LastUpdateEpoch:  int64(ds.LastUpdatedEpoch),
				SlashEpoch:       int64(ds.SlashEpoch),
				StateRoot:        ec.CurrTs.ParentState().String(),
			})
			return nil
		}); err != nil {
			return nil, fmt.Errorf("walking current deal states: %w", err)
		}
		return out, nil
	}

	changed, err := ec.CurrState.StatesChanged(ec.PrevState)
	if err != nil {
		return nil, fmt.Errorf("checking for deal state changes: %w", err)
	}

	if !changed {
		return nil, nil
	}

	changes, err := market.DiffDealStates(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, fmt.Errorf("diffing deal states: %w", err)
	}

	out := make(marketmodel.MarketDealStates, len(changes.Added)+len(changes.Modified))
	idx := 0
	for _, add := range changes.Added {
		out[idx] = &marketmodel.MarketDealState{
			Height:           int64(ec.CurrTs.Height()),
			DealID:           uint64(add.ID),
			SectorStartEpoch: int64(add.Deal.SectorStartEpoch),
			LastUpdateEpoch:  int64(add.Deal.LastUpdatedEpoch),
			SlashEpoch:       int64(add.Deal.SlashEpoch),
			StateRoot:        ec.CurrTs.ParentState().String(),
		}
		idx++
	}
	for _, mod := range changes.Modified {
		out[idx] = &marketmodel.MarketDealState{
			Height:           int64(ec.CurrTs.Height()),
			DealID:           uint64(mod.ID),
			SectorStartEpoch: int64(mod.To.SectorStartEpoch),
			LastUpdateEpoch:  int64(mod.To.LastUpdatedEpoch),
			SlashEpoch:       int64(mod.To.SlashEpoch),
			StateRoot:        ec.CurrTs.ParentState().String(),
		}
		idx++
	}
	return out, nil
}
