package v7

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/util"

	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v1"

	miner "github.com/filecoin-project/specs-actors/v7/actors/builtin/miner"
)

type SectorDeal struct{}

func (s SectorDeal) Transform(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *minerdiff.StateDiffResult) (model.Persistable, error) {
	sectors := change.SectorChanges
	out := minermodel.MinerSectorDealList{}
	height := int64(current.Height())
	minerAddr := addr.String()
	for _, sector := range sectors {
		switch sector.Change {
		case core.ChangeTypeAdd:
			s := new(miner.SectorOnChainInfo)
			if err := s.UnmarshalCBOR(bytes.NewReader(sector.Current.Raw)); err != nil {
				return nil, err
			}
			for _, deal := range s.DealIDs {
				out = append(out, &minermodel.MinerSectorDeal{
					Height:   height,
					MinerID:  minerAddr,
					SectorID: uint64(s.SectorNumber),
					DealID:   uint64(deal),
				})
			}
		case core.ChangeTypeModify:
			previousSector := new(miner.SectorOnChainInfo)
			if err := previousSector.UnmarshalCBOR(bytes.NewReader(sector.Previous.Raw)); err != nil {
				return nil, err
			}
			currentSector := new(miner.SectorOnChainInfo)
			if err := currentSector.UnmarshalCBOR(bytes.NewReader(sector.Current.Raw)); err != nil {
				return nil, err
			}
			for _, deal := range util.CompareDealIDs(currentSector.DealIDs, previousSector.DealIDs) {
				out = append(out, &minermodel.MinerSectorDeal{
					Height:   height,
					MinerID:  minerAddr,
					SectorID: uint64(currentSector.SectorNumber),
					DealID:   uint64(deal),
				})
			}
		}
	}
	return out, nil
}
