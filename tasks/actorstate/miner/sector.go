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

	sectorChanges, err := node.DiffSectors(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	// transform sector changes to a model
	sectorModel := minermodel.MinerSectorInfoV1_6List{}
	for _, added := range sectorChanges.Added {
		sm := &minermodel.MinerSectorInfoV1_6{
			Height:                int64(ec.CurrTs.Height()),
			MinerID:               a.Address.String(),
			SectorID:              uint64(added.SectorNumber),
			StateRoot:             a.Current.ParentState().String(),
			SealedCID:             added.SealedCID.String(),
			ActivationEpoch:       int64(added.Activation),
			ExpirationEpoch:       int64(added.Expiration),
			DealWeight:            added.DealWeight.String(),
			VerifiedDealWeight:    added.VerifiedDealWeight.String(),
			InitialPledge:         added.InitialPledge.String(),
			ExpectedDayReward:     added.ExpectedDayReward.String(),
			ExpectedStoragePledge: added.ExpectedStoragePledge.String(),
		}
		sectorModel = append(sectorModel, sm)
	}

	// do the same for extended sectors, since they have a new deadline
	for _, extended := range sectorChanges.Extended {
		sm := &minermodel.MinerSectorInfoV1_6{
			Height:                int64(ec.CurrTs.Height()),
			MinerID:               a.Address.String(),
			SectorID:              uint64(extended.To.SectorNumber),
			StateRoot:             a.Current.ParentState().String(),
			SealedCID:             extended.To.SealedCID.String(),
			ActivationEpoch:       int64(extended.To.Activation),
			ExpirationEpoch:       int64(extended.To.Expiration),
			DealWeight:            extended.To.DealWeight.String(),
			VerifiedDealWeight:    extended.To.VerifiedDealWeight.String(),
			InitialPledge:         extended.To.InitialPledge.String(),
			ExpectedDayReward:     extended.To.ExpectedDayReward.String(),
			ExpectedStoragePledge: extended.To.ExpectedStoragePledge.String(),
		}
		sectorModel = append(sectorModel, sm)
	}

	return sectorModel, nil
}
