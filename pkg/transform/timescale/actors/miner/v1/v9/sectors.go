package v9

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"

	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v1"

	miner "github.com/filecoin-project/go-state-types/builtin/v9/miner"
)

type Sector struct {
}

func (s Sector) Transform(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *minerdiff.StateDiffResult) (model.Persistable, error) {
	var sectors []*miner.SectorOnChainInfo
	changes := change.SectorChanges
	for _, sector := range changes {
		// only care about modified and added sectors
		if sector.Change == core.ChangeTypeRemove {
			continue
		}
		s := new(miner.SectorOnChainInfo)
		if err := s.UnmarshalCBOR(bytes.NewReader(sector.Current.Raw)); err != nil {
			return nil, err
		}
		sectors = append(sectors, s)
	}
	return MinerSectorChangesAsModel(ctx, current, addr, sectors)
}

func MinerSectorChangesAsModel(ctx context.Context, current *types.TipSet, addr address.Address, sectors []*miner.SectorOnChainInfo) (model.Persistable, error) {
	sectorModel := make(minermodel.MinerSectorInfoV7List, len(sectors))
	for i, sector := range sectors {
		sectorKeyCID := ""
		if sector.SectorKeyCID != nil {
			sectorKeyCID = sector.SectorKeyCID.String()
		}
		sectorModel[i] = &minermodel.MinerSectorInfoV7{
			Height:                int64(current.Height()),
			MinerID:               addr.String(),
			StateRoot:             current.ParentState().String(),
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
