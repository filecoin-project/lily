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

	log.Infof("loading sectors for miner %s, height: %v", a.Address.String(), a.Current.Height())

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
			log.Errorf("diffing sectors for miner %s, height: %v: %v", a.Address.String(), a.Current.Height(), err)
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

		replacedDayReward := sector.ReplacedDayReward
		replacedDayRewardStr := "0"
		if !replacedDayReward.Nil() {
			replacedDayRewardStr = replacedDayReward.String()
		}

		expectedDayReward := sector.ExpectedDayReward
		expectedDayRewardStr := "0"
		if !expectedDayReward.Nil() {
			expectedDayRewardStr = expectedDayReward.String()
		}

		expectedStoragePledge := sector.ExpectedStoragePledge
		expectedStoragePledgeStr := "0"
		if !expectedStoragePledge.Nil() {
			expectedStoragePledgeStr = expectedStoragePledge.String()
		}

		// Daily Fee
		dailyFee := sector.DailyFee
		dailyFeeStr := "0"
		if !dailyFee.Nil() {
			dailyFeeStr = dailyFee.String()
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
			ExpectedDayReward:     expectedDayRewardStr,
			ExpectedStoragePledge: expectedStoragePledgeStr,
			ReplacedDayReward:     replacedDayRewardStr,
			PowerBaseEpoch:        int64(sector.PowerBaseEpoch),
			SectorKeyCID:          sectorKeyCID,
			DailyFee:              dailyFeeStr,
		}
	}

	return sectorModel, nil
}

func (V7SectorInfoExtractor) Transform(_ context.Context, data model.PersistableList) (model.PersistableList, error) {
	persistableList := make(minermodel.MinerSectorInfoV7List, 0, len(data))
	for _, d := range data {
		ml, ok := d.(minermodel.MinerSectorInfoV7List)
		if !ok {
			return nil, fmt.Errorf("expected MinerSectorInfoV7 type but got: %T", d)
		}
		for _, m := range ml {
			persistableList = append(persistableList, m)
		}
	}
	return model.PersistableList{persistableList}, nil
}
