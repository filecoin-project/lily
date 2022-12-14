package minerdiff

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-bitfield"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/pkg/extract/actors"
	"github.com/filecoin-project/lily/tasks"
)

var _ actors.ActorStateChange = (*SectorStatusChange)(nil)

type SectorStatusChange struct {
	// Removed sectors this epoch
	Removed bitfield.BitField `cborgen:"removed"`
	// Recovering sectors this epoch
	Recovering bitfield.BitField `cborgen:"recovering"`
	// Faulted sectors this epoch
	Faulted bitfield.BitField `cborgen:"faulted"`
	// Recovered sectors this epoch
	Recovered bitfield.BitField `cborgen:"recovered"`
}

const KindMinerSectorStatus = "miner_sector_status"

func (s SectorStatusChange) Kind() actors.ActorStateKind {
	return KindMinerSectorStatus
}

type SectorStatus struct{}

func (SectorStatus) Diff(ctx context.Context, api tasks.DataSource, act *actors.ActorChange) (actors.ActorStateChange, error) {
	start := time.Now()
	defer func() {
		log.Debugw("Diff", "kind", KindMinerSectorStatus, zap.Inline(act), "duration", time.Since(start))
	}()
	child, err := api.MinerLoad(api.Store(), act.Current)
	if err != nil {
		return nil, err
	}
	parent, err := api.MinerLoad(api.Store(), act.Executed)
	if err != nil {
		return nil, err
	}
	return DiffMinerSectorStates(ctx, child, parent)
}

// DiffMinerSectorStates loads the SectorStates for the current and parent miner states in parallel from `extState`.
// Then compares current and parent SectorStates to produce a SectorStateEvents structure containing all sectors that are
// removed, recovering, faulted, and recovered for the state transition from parent miner state to current miner state.
func DiffMinerSectorStates(ctx context.Context, child, parent miner.State) (*SectorStatusChange, error) {
	var (
		previous, current *SectorStates
		err               error
	)

	// load previous and current miner sector states in parallel
	grp, grpCtx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		previous, err = LoadSectorState(grpCtx, parent)
		if err != nil {
			return fmt.Errorf("loading previous sector states %w", err)
		}
		return nil
	})
	grp.Go(func() error {
		current, err = LoadSectorState(grpCtx, child)
		if err != nil {
			return fmt.Errorf("loading current sector states %w", err)
		}
		return nil
	})
	// if either load operation fails abort
	if err := grp.Wait(); err != nil {
		return nil, err
	}

	// previous live sector minus current live sectors are sectors removed this epoch.
	removed, err := bitfield.SubtractBitField(previous.Live, current.Live)
	if err != nil {
		return nil, fmt.Errorf("comparing previous live sectors to current live sectors %w", err)
	}

	// current recovering sectors minus previous recovering sectors are sectors recovering this epoch.
	recovering, err := bitfield.SubtractBitField(current.Recovering, previous.Recovering)
	if err != nil {
		return nil, fmt.Errorf("comparing current recovering sectors to previous recovering sectors %w", err)
	}

	// current faulty sectors minus previous faulty sectors are sectors faulted this epoch.
	faulted, err := bitfield.SubtractBitField(current.Faulty, previous.Faulty)
	if err != nil {
		return nil, fmt.Errorf("comparing current faulty sectors to previous faulty sectors %w", err)
	}

	// previous faulty sectors matching (intersect) active sectors are sectors recovered this epoch.
	recovered, err := bitfield.IntersectBitField(previous.Faulty, current.Active)
	if err != nil {
		return nil, fmt.Errorf("comparing previous faulty sectors to current active sectors %w", err)
	}

	return &SectorStatusChange{
		Removed:    removed,
		Recovering: recovering,
		Faulted:    faulted,
		Recovered:  recovered,
	}, nil

}

// SectorStates contains a set of bitfields for active, live, fault, and recovering sectors.
type SectorStates struct {
	// Active sectors are those that are neither terminated nor faulty nor unproven, i.e. actively contributing power.
	Active bitfield.BitField
	// Live sectors are those that are not terminated (but may be faulty).
	Live bitfield.BitField
	// Faulty contains a subset of sectors detected/declared faulty and not yet recovered (excl. from PoSt).
	Faulty bitfield.BitField
	// Recovering contains a subset of faulty sectors expected to recover on next PoSt.
	Recovering bitfield.BitField
}

// LoadSectorState loads all sectors from a miners partitions and returns a SectorStates structure containing individual
// bitfields for all active, live, faulty and recovering sector.
func LoadSectorState(ctx context.Context, state miner.State) (*SectorStates, error) {
	activeSectors := []bitfield.BitField{}
	liveSectors := []bitfield.BitField{}
	faultySectors := []bitfield.BitField{}
	recoveringSectors := []bitfield.BitField{}

	// iterate the sector states
	if err := state.ForEachDeadline(func(_ uint64, dl miner.Deadline) error {
		return dl.ForEachPartition(func(_ uint64, part miner.Partition) error {
			active, err := part.ActiveSectors()
			if err != nil {
				return err
			}
			activeSectors = append(activeSectors, active)

			live, err := part.LiveSectors()
			if err != nil {
				return err
			}
			liveSectors = append(liveSectors, live)

			faulty, err := part.FaultySectors()
			if err != nil {
				return err
			}
			faultySectors = append(faultySectors, faulty)

			recovering, err := part.RecoveringSectors()
			if err != nil {
				return err
			}
			recoveringSectors = append(recoveringSectors, recovering)

			return nil
		})
	}); err != nil {
		return nil, err
	}
	var err error
	sectorStates := &SectorStates{}
	if sectorStates.Active, err = bitfield.MultiMerge(activeSectors...); err != nil {
		return nil, err
	}
	if sectorStates.Live, err = bitfield.MultiMerge(liveSectors...); err != nil {
		return nil, err
	}
	if sectorStates.Faulty, err = bitfield.MultiMerge(faultySectors...); err != nil {
		return nil, err
	}
	if sectorStates.Recovering, err = bitfield.MultiMerge(recoveringSectors...); err != nil {
		return nil, err
	}

	return sectorStates, nil
}
