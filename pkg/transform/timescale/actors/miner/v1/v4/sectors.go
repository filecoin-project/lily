package v4

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	minertypes "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/types"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"

	miner "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
)

type Sector struct{}

func (s Sector) Transform(ctx context.Context, current, executed *types.TipSet, miners []*minertypes.MinerStateChange) model.Persistable {
	report := data.StartProcessingReport(tasktype.MinerSectorInfoV1_6, current)
	for _, m := range miners {
		var sectors []*miner.SectorOnChainInfo
		changes := m.StateChange.SectorChanges
		for _, sector := range changes {
			// only care about modified and added sectors
			if sector.Change == core.ChangeTypeRemove {
				continue
			}
			s := new(miner.SectorOnChainInfo)
			if err := s.UnmarshalCBOR(bytes.NewReader(sector.Current.Raw)); err != nil {
				report.AddError(err)
				continue
			}
			sectors = append(sectors, s)
		}
		report.AddModels(MinerSectorChangesAsModel(ctx, current, m.Address, sectors))
	}
	return report.Finish()
}

func MinerSectorChangesAsModel(ctx context.Context, current *types.TipSet, addr address.Address, sectors []*miner.SectorOnChainInfo) model.Persistable {
	sectorModel := make(minermodel.MinerSectorInfoV1_6List, len(sectors))
	for i, sector := range sectors {
		sectorModel[i] = &minermodel.MinerSectorInfoV1_6{
			Height:                int64(current.Height()),
			MinerID:               addr.String(),
			SectorID:              uint64(sector.SectorNumber),
			StateRoot:             current.ParentState().String(),
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

	return sectorModel
}
