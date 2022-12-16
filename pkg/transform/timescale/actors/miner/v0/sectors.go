package v0

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
)

func HandleMinerSectorChanges(ctx context.Context, store adt.Store, current, executed *types.TipSet, addr address.Address, changes v0.SectorChangeList) (model.Persistable, error) {
	var sectors []*miner0.SectorOnChainInfo
	for _, change := range changes {
		// only care about modified and added sectors
		if change.Change == core.ChangeTypeRemove {
			continue
		}
		// change.Current is the newly added sector, or its state after modification.
		if err := core.StateReadDeferred(ctx, change.Current, func(sector *miner0.SectorOnChainInfo) error {
			sectors = append(sectors, sector)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return MinerSectorChangesAsModel(ctx, current, addr, sectors)
}

func MinerSectorChangesAsModel(ctx context.Context, current *types.TipSet, addr address.Address, sectors []*miner0.SectorOnChainInfo) (model.Persistable, error) {
	sectorModel := make(minermodel.MinerSectorInfoV1_6List, len(sectors))
	for i, sector := range sectors {
		sectorModel[i] = &minermodel.MinerSectorInfoV1_6{
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
		}
	}

	return sectorModel, nil
}
