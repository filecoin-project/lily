package miner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type SectorDealsExtractor struct{}

func (SectorDealsExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "SectorDealsExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "SectorDealsExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	var sectors []*miner.SectorOnChainInfo
	if !ec.HasPreviousState() {
		// If the miner doesn't have previous state list all of its current sectors.
		sectors, err = ec.CurrState.LoadSectors(nil)
		if err != nil {
			return nil, fmt.Errorf("loading miner sectors: %w", err)
		}
	} else {
		// If the miner has previous state compute the list of new sectors in its current state.
		sectorChanges, err := node.DiffSectors(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}
		// sectors that were added _may_ contain dealIDs'
		for i := range sectorChanges.Added {
			sectors = append(sectors, &sectorChanges.Added[i])
		}
		// sectors that were snapped _will_ contain dealID's per: https://github.com/filecoin-project/FIPs/blob/master/FIPS/fip-0019.md#spec
		for i := range sectorChanges.Snapped {
			sectors = append(sectors, &sectorChanges.Snapped[i].To)
		}
	}

	sectorDealsModel := minermodel.MinerSectorDealList{}
	for _, sector := range sectors {
		for _, dealID := range sector.DealIDs {
			sectorDealsModel = append(sectorDealsModel, &minermodel.MinerSectorDeal{
				Height:   int64(ec.CurrTs.Height()),
				MinerID:  a.Address.String(),
				SectorID: uint64(sector.SectorNumber),
				DealID:   uint64(dealID),
			})
		}
	}
	return sectorDealsModel, nil
}
