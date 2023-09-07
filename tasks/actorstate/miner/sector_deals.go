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
	"github.com/filecoin-project/lily/tasks/actorstate/miner/extraction"
)

type SectorDealsExtractor struct{}

func (SectorDealsExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "SectorDealsExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "SectorDealsExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := extraction.LoadMinerStates(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	return ExtractSectorDealsModel(ctx, ec)
}

func ExtractSectorDealsModel(ctx context.Context, ec extraction.State) (minermodel.MinerSectorDealList, error) {
	var (
		result  minermodel.MinerSectorDealList
		sectors []*miner.SectorOnChainInfo
		err     error
	)
	if ec.ParentState() == nil {
		// If the miner doesn't have previous state list all of its current sectors.
		sectors, err = ec.CurrentState().LoadSectors(nil)
		if err != nil {
			return nil, fmt.Errorf("loading miner sectors: %w", err)
		}
	} else {
		// If the miner has previous state compute the list of new sectors in its current state.
		sectorChanges, err := ec.API().DiffSectors(ctx, ec.Address(), ec.CurrentTipSet(), ec.ParentTipSet(), ec.ParentState(), ec.CurrentState())
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

	for _, sector := range sectors {
		for _, dealID := range sector.DealIDs {
			result = append(result, &minermodel.MinerSectorDeal{
				Height:   int64(ec.CurrentTipSet().Height()),
				MinerID:  ec.Address().String(),
				SectorID: uint64(sector.SectorNumber),
				DealID:   uint64(dealID),
			})
		}
	}
	return result, nil
}

func (SectorDealsExtractor) Transform(_ context.Context, data model.PersistableList) (model.PersistableList, error) {
	persistableList := make(minermodel.MinerSectorDealList, 0, len(data))
	for _, d := range data {
		ml, ok := d.(minermodel.MinerSectorDealList)
		if !ok {
			return nil, fmt.Errorf("expected MinerSectorDealList type but got: %T", d)
		}
		for _, m := range ml {
			persistableList = append(persistableList, m)
		}
	}
	return model.PersistableList{persistableList}, nil
}
