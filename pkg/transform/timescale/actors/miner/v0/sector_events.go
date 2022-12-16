package v0

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v0"
)

func HandleMinerSectorEvents(ctx context.Context, store adt.Store, current, executed *types.TipSet, addr address.Address, precommits v0.PreCommitChangeList, sectors v0.SectorChangeList, sectorstatus *v0.SectorStatusChange) (model.Persistable, error) {
	out := minermodel.MinerSectorEventList{}
	height := int64(current.Height())
	minerAddr := addr.String()
	stateRoot := current.ParentState().String()
	for _, precommit := range precommits {
		// only care about new precommits
		if precommit.Change != core.ChangeTypeAdd {
			continue
		}
		sectorID, err := abi.ParseUIntKey(string(precommit.SectorNumber))
		if err != nil {
			return nil, err
		}
		out = append(out, &minermodel.MinerSectorEvent{
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
			if err := core.StateReadDeferred(ctx, sector.Current, func(s *miner0.SectorOnChainInfo) error {
				if len(s.DealIDs) == 0 {
					event = minermodel.CommitCapacityAdded
				}
				return nil
			}); err != nil {
				return nil, err
			}
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    height,
				MinerID:   minerAddr,
				SectorID:  sector.SectorNumber,
				StateRoot: stateRoot,
				Event:     event,
			})
		case core.ChangeTypeModify:
			previousSector := new(miner0.SectorOnChainInfo)
			if err := previousSector.UnmarshalCBOR(bytes.NewReader(sector.Previous.Raw)); err != nil {
				return nil, err
			}
			currentSector := new(miner0.SectorOnChainInfo)
			if err := currentSector.UnmarshalCBOR(bytes.NewReader(sector.Current.Raw)); err != nil {
				return nil, err
			}
			if previousSector.Expiration != currentSector.Expiration {
				out = append(out, &minermodel.MinerSectorEvent{
					Height:    height,
					MinerID:   minerAddr,
					SectorID:  sector.SectorNumber,
					StateRoot: stateRoot,
					Event:     minermodel.SectorExtended,
				})
			}
			// snapping didn't exist at this epoch
			/*
				if previousSector.SectorKeyCID == nil && currentSector.SectorKeyCID != nil {
					out = append(out, &minermodel.MinerSectorEvent{
						Height:    height,
						MinerID:   minerAddr,
						SectorID:  sector.SectorNumber,
						StateRoot: stateRoot,
						Event:     minermodel.SectorSnapped,
					})
				}

			*/
		}
	}
	if sectorstatus == nil {
		return out, nil
	}
	// all sectors removed this epoch are considered terminated, this includes both early terminations and expirations.
	if err := sectorstatus.Removed.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    height,
			MinerID:   minerAddr,
			SectorID:  u,
			StateRoot: stateRoot,
			Event:     minermodel.SectorTerminated,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if err := sectorstatus.Recovering.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    height,
			MinerID:   minerAddr,
			SectorID:  u,
			StateRoot: stateRoot,
			Event:     minermodel.SectorRecovering,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if err := sectorstatus.Faulted.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    height,
			MinerID:   minerAddr,
			SectorID:  u,
			StateRoot: stateRoot,
			Event:     minermodel.SectorFaulted,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if err := sectorstatus.Recovered.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    height,
			MinerID:   minerAddr,
			SectorID:  u,
			StateRoot: stateRoot,
			Event:     minermodel.SectorRecovered,
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}
