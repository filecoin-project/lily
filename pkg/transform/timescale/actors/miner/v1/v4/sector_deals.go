package v4

import (
	"bytes"
	"context"

	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/util"
	minertypes "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/types"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"

	miner "github.com/filecoin-project/specs-actors/v4/actors/builtin/miner"
)

type SectorDeal struct{}

func (s SectorDeal) Transform(ctx context.Context, current, executed *types.TipSet, miners []*minertypes.MinerStateChange) model.Persistable {
	report := data.StartProcessingReport(tasktype.MinerSectorDeal, current)
	for _, m := range miners {

		sectors := m.StateChange.SectorChanges
		height := int64(current.Height())
		minerAddr := m.Address.String()

		for _, sector := range sectors {
			switch sector.Change {
			case core.ChangeTypeAdd:
				s := new(miner.SectorOnChainInfo)
				if err := s.UnmarshalCBOR(bytes.NewReader(sector.Current.Raw)); err != nil {
					report.AddError(err)
					continue
				}
				for _, deal := range s.DealIDs {
					report.AddModels(&minermodel.MinerSectorDeal{
						Height:   height,
						MinerID:  minerAddr,
						SectorID: uint64(s.SectorNumber),
						DealID:   uint64(deal),
					})
				}
			case core.ChangeTypeModify:
				previousSector := new(miner.SectorOnChainInfo)
				if err := previousSector.UnmarshalCBOR(bytes.NewReader(sector.Previous.Raw)); err != nil {
					report.AddError(err)
					continue
				}
				currentSector := new(miner.SectorOnChainInfo)
				if err := currentSector.UnmarshalCBOR(bytes.NewReader(sector.Current.Raw)); err != nil {
					report.AddError(err)
					continue
				}
				for _, deal := range util.CompareDealIDs(currentSector.DealIDs, previousSector.DealIDs) {
					report.AddModels(&minermodel.MinerSectorDeal{
						Height:   height,
						MinerID:  minerAddr,
						SectorID: uint64(currentSector.SectorNumber),
						DealID:   uint64(deal),
					})
				}
			}
		}
	}
	return report.Finish()
}
