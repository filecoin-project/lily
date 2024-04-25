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
	"github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
	logging "github.com/ipfs/go-log/v2"
)

var _ actorstate.ActorStateExtractor = (*SectorDealStateExtractor)(nil)

type SectorDealStateExtractor struct{}

var logger = logging.Logger("lily/tasks/sector_deal_logger")

func (SectorDealStateExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "SectorDealStateExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "SectorDealStateExtractor.Extract")
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
				SectorStartEpoch: int64(ds.SectorStartEpoch()),
				LastUpdateEpoch:  int64(ds.LastUpdatedEpoch()),
				SlashEpoch:       int64(ds.SlashEpoch()),
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

	result, err := ec.CurrState.GetProviderSectors()
	dealSectorMap := make(map[abi.DealID]abi.SectorID)
	for sectorId, dealIDs := range result {
		for _, dealID := range dealIDs {
			dealSectorMap[dealID] = sectorId
		}
	}

	changes, err := market.DiffDealStates(ctx, ec.Store, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, fmt.Errorf("diffing deal states: %w", err)
	}

	out := make(miner.MinerSectorDealListV2, len(changes.Added)+len(changes.Modified))
	idx := 0
	for _, add := range changes.Added {
		out[idx] = &miner.MinerSectorDealV2{
			Height:   int64(ec.CurrTs.Height()),
			DealID:   uint64(add.ID),
			SectorID: uint64(dealSectorMap[add.ID].Number),
			MinerID:  dealSectorMap[add.ID].Miner.String(),
		}
		idx++
	}
	for _, mod := range changes.Modified {
		out[idx] = &miner.MinerSectorDealV2{
			Height:   int64(ec.CurrTs.Height()),
			DealID:   uint64(mod.ID),
			SectorID: uint64(dealSectorMap[mod.ID].Number),
			MinerID:  dealSectorMap[mod.ID].Miner.String(),
		}
		idx++
	}
	return out, nil
}
