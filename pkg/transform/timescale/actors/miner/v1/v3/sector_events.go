package v3

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	minertypes "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/types"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"

	miner "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
)

type SectorEvent struct{}

func (s SectorEvent) Transform(ctx context.Context, current, executed *types.TipSet, miners []*minertypes.MinerStateChange) model.Persistable {
	report := data.StartProcessingReport(tasktype.MinerSectorEvent, current)
	for _, m := range miners {
		var (
			precommits   = m.StateChange.PreCommitChanges
			sectors      = m.StateChange.SectorChanges
			sectorstatus = m.StateChange.SectorStatusChanges
			height       = int64(current.Height())
			minerAddr    = m.Address.String()
			stateRoot    = current.ParentState().String()
		)
		for _, precommit := range precommits {
			// only care about new precommits
			if precommit.Change != core.ChangeTypeAdd {
				continue
			}
			sectorID, err := abi.ParseUIntKey(string(precommit.SectorNumber))
			if err != nil {
				report.AddError(err)
				continue
			}
			report.AddModels(&minermodel.MinerSectorEvent{
				Height:    height,
				MinerID:   minerAddr,
				SectorID:  sectorID,
				StateRoot: stateRoot,
				Event:     minermodel.PreCommitAdded,
			})
		}
		for _, sector := range sectors {
			switch sector.Change {
			case core.ChangeTypeAdd:
				event := minermodel.SectorAdded
				s := new(miner.SectorOnChainInfo)
				if err := s.UnmarshalCBOR(bytes.NewReader(sector.Current.Raw)); err != nil {
					report.AddError(err)
					continue
				}
				if len(s.DealIDs) == 0 {
					event = minermodel.CommitCapacityAdded
				}
				report.AddModels(&minermodel.MinerSectorEvent{
					Height:    height,
					MinerID:   minerAddr,
					SectorID:  sector.SectorNumber,
					StateRoot: stateRoot,
					Event:     event,
				})
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
				if previousSector.Expiration != currentSector.Expiration {
					report.AddModels(&minermodel.MinerSectorEvent{
						Height:    height,
						MinerID:   minerAddr,
						SectorID:  sector.SectorNumber,
						StateRoot: stateRoot,
						Event:     minermodel.SectorExtended,
					})
				}
			}
		}
		if sectorstatus == nil {
			continue
		}
		// all sectors removed this epoch are considered terminated, this includes both early terminations and expirations.
		if err := sectorstatus.Removed.ForEach(func(u uint64) error {
			report.AddModels(&minermodel.MinerSectorEvent{
				Height:    height,
				MinerID:   minerAddr,
				SectorID:  u,
				StateRoot: stateRoot,
				Event:     minermodel.SectorTerminated,
			})
			return nil
		}); err != nil {
			report.AddError(err)
			continue
		}

		if err := sectorstatus.Recovering.ForEach(func(u uint64) error {
			report.AddModels(&minermodel.MinerSectorEvent{
				Height:    height,
				MinerID:   minerAddr,
				SectorID:  u,
				StateRoot: stateRoot,
				Event:     minermodel.SectorRecovering,
			})
			return nil
		}); err != nil {
			report.AddError(err)
			continue
		}

		if err := sectorstatus.Faulted.ForEach(func(u uint64) error {
			report.AddModels(&minermodel.MinerSectorEvent{
				Height:    height,
				MinerID:   minerAddr,
				SectorID:  u,
				StateRoot: stateRoot,
				Event:     minermodel.SectorFaulted,
			})
			return nil
		}); err != nil {
			report.AddError(err)
			continue
		}

		if err := sectorstatus.Recovered.ForEach(func(u uint64) error {
			report.AddModels(&minermodel.MinerSectorEvent{
				Height:    height,
				MinerID:   minerAddr,
				SectorID:  u,
				StateRoot: stateRoot,
				Event:     minermodel.SectorRecovered,
			})
			return nil
		}); err != nil {
			report.AddError(err)
			continue
		}
	}
	return report.Finish()
}
