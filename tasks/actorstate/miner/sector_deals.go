package miner

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

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
		return nil, xerrors.Errorf("creating miner state extraction context: %w", err)
	}

	sectorDealsModel := minermodel.MinerSectorDealList{}
	sectorChanges, err := node.DiffSectors(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	for _, newSector := range sectorChanges.Added {
		for _, dealID := range newSector.DealIDs {
			sectorDealsModel = append(sectorDealsModel, &minermodel.MinerSectorDeal{
				Height:   int64(ec.CurrTs.Height()),
				MinerID:  a.Address.String(),
				SectorID: uint64(newSector.SectorNumber),
				DealID:   uint64(dealID),
			})
		}
	}
	return sectorDealsModel, nil
}
