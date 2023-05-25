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

type V7SectorInfoExtractor struct{}

func (V7SectorInfoExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "V7SectorInfoExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "V7SectorInfoExtractor.Extract")
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
		for i := range sectorChanges.Added {
			sectors = append(sectors, &sectorChanges.Added[i])
		}
		for i := range sectorChanges.Extended {
			sectors = append(sectors, &sectorChanges.Extended[i].To)
		}
		for i := range sectorChanges.Snapped {
			sectors = append(sectors, &sectorChanges.Snapped[i].To)
		}
	}
	sectorModel := make(minermodel.MinerSectorInfoV7List, len(sectors))
	for i, sector := range sectors {
		sectorKeyCID := ""
		if sector.SectorKeyCID != nil {
			sectorKeyCID = sector.SectorKeyCID.String()
		}
		sectorModel[i] = &minermodel.MinerSectorInfoV7{
			Height:                int64(a.Current.Height()),
			MinerID:               a.Address.String(),
			StateRoot:             a.Current.ParentState().String(),
			SectorID:              uint64(sector.SectorNumber),
			SealedCID:             sector.SealedCID.String(),
			ActivationEpoch:       int64(sector.Activation),
			ExpirationEpoch:       int64(sector.Expiration),
			DealWeight:            sector.DealWeight.String(),
			VerifiedDealWeight:    sector.VerifiedDealWeight.String(),
			InitialPledge:         sector.InitialPledge.String(),
			ExpectedDayReward:     sector.ExpectedDayReward.String(),
			ExpectedStoragePledge: sector.ExpectedStoragePledge.String(),
			SectorKeyCID:          sectorKeyCID,
		}
	}

	return sectorModel, nil
}

func (V7SectorInfoExtractor) Transform(ctx context.Context, data model.PersistableList) (model.PersistableList, error) {
	persistableList := make(minermodel.MinerSectorInfoV7List, 0, len(data))
	for _, d := range data {
		a, ok := d.(*minermodel.MinerSectorInfoV7)
		if !ok {
			return nil, fmt.Errorf("expected MinerSectorInfoV7 type but got: %T", d)
		}
		persistableList = append(persistableList, a)
	}
	return model.PersistableList{persistableList}, nil
}
