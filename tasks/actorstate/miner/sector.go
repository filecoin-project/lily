package miner

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type SectorInfoExtractor struct{}

func (SectorInfoExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "SectorInfoExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "SectorInfoExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("creating miner state extraction context: %w", err)
	}

	var sectors []*miner.SectorOnChainInfo
	if !ec.HasPreviousState() {
		// If the miner doesn't have previous state list all of its current sectors.
		sectors, err = ec.CurrState.LoadSectors(nil)
		if err != nil {
			return nil, xerrors.Errorf("loading miner sectors: %w", err)
		}
	} else {
		// If the miner has previous state compute the list of sectors added in its new state.
		sectorChanges, err := node.DiffSectors(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}
		for _, sector := range sectorChanges.Added {
			sectors = append(sectors, &sector)
		}
		for _, sector := range sectorChanges.Extended {
			sectors = append(sectors, &sector.To)
		}
	}

	sectorModel := make(minermodel.MinerSectorInfoV1_6List, len(sectors))
	for i, sector := range sectors {
		sectorModel[i] = &minermodel.MinerSectorInfoV1_6{
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
		}
	}

	return sectorModel, nil
}
