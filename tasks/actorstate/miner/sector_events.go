package miner

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-bitfield"
	minertypes "github.com/filecoin-project/go-state-types/builtin/v9/miner"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/lily/chain/actors"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lily/tasks/actorstate/miner/extraction"
)

type SectorEventsExtractor struct{}

func (SectorEventsExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "SectorEventsExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "SectorEventsExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	extState, err := extraction.LoadMinerStates(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	var (
		sectorChanges      = miner.MakeSectorChanges()
		preCommitChanges   = miner.MakePreCommitChanges()
		sectorStateChanges = &SectorStateEvents{
			Removed:    bitfield.New(),
			Recovering: bitfield.New(),
			Faulted:    bitfield.New(),
			Recovered:  bitfield.New(),
		}
	)
	if extState.ParentState() == nil {
		// If the miner doesn't have previous state list all of its current sectors and precommits
		sectors, err := extState.CurrentState().LoadSectors(nil)
		if err != nil {
			return nil, fmt.Errorf("loading miner sectors: %w", err)
		}

		for _, sector := range sectors {
			sectorChanges.Added = append(sectorChanges.Added, *sector)
		}

		if err = extState.CurrentState().ForEachPrecommittedSector(func(info minertypes.SectorPreCommitOnChainInfo) error {
			preCommitChanges.Added = append(preCommitChanges.Added, info)
			return nil
		}); err != nil {
			return nil, err
		}

	} else {
		// If the miner has previous state compute the list of new sectors and precommit in its current state.
		grp, grpCtx := errgroup.WithContext(ctx)
		grp.Go(func() error {
			start := time.Now()
			// collect changes made to miner precommit map (HAMT)
			if extState.CurrentState().ActorVersion() > actors.Version8 {
				preCommitChanges, err = node.DiffPreCommits(grpCtx, a.Address, a.Current, a.Executed, extState.ParentState(), extState.CurrentState())
				if err != nil {
					return fmt.Errorf("diffing precommits %w", err)
				}
			} else {
				var v8PreCommitChanges *miner.PreCommitChangesV8
				v8PreCommitChanges, err = node.DiffPreCommitsV8(grpCtx, a.Address, a.Current, a.Executed, extState.ParentState(), extState.CurrentState())
				if err != nil {
					return fmt.Errorf("diffing precommits: %w", err)
				}
				for _, c := range v8PreCommitChanges.Added {
					preCommitChanges.Added = append(preCommitChanges.Added, minertypes.SectorPreCommitOnChainInfo{
						Info: minertypes.SectorPreCommitInfo{
							SealProof:     c.Info.SealProof,
							SectorNumber:  c.Info.SectorNumber,
							SealedCID:     c.Info.SealedCID,
							SealRandEpoch: c.Info.SealRandEpoch,
							DealIDs:       nil,
							Expiration:    c.Info.Expiration,
							UnsealedCid:   nil,
						},
						PreCommitDeposit: c.PreCommitDeposit,
						PreCommitEpoch:   c.PreCommitEpoch,
					})
				}
			}

			log.Debugw("diff precommits", "miner", a.Address, "duration", time.Since(start))
			return nil
		})
		grp.Go(func() error {
			start := time.Now()
			// collect changes made to miner sector array (AMT)
			sectorChanges, err = node.DiffSectors(grpCtx, a.Address, a.Current, a.Executed, extState.ParentState(), extState.CurrentState())
			if err != nil {
				return fmt.Errorf("diffing sectors %w", err)
			}
			log.Debugw("diff sectors", "miner", a.Address, "duration", time.Since(start))
			return nil
		})
		grp.Go(func() error {
			start := time.Now()
			// collect changes made to miner sectors across all miner partition states
			sectorStateChanges, err = DiffMinerSectorStates(grpCtx, extState)
			if err != nil {
				return fmt.Errorf("diffing sector states %w", err)
			}
			log.Debugw("diff sector states", "miner", a.Address, "duration", time.Since(start))
			return nil
		})
		if err := grp.Wait(); err != nil {
			return nil, err
		}
	}

	// transform the sector events to a model.
	sectorEventModel, err := ExtractSectorEvents(extState, sectorChanges, preCommitChanges, sectorStateChanges)
	if err != nil {
		return nil, err
	}

	return sectorEventModel, nil
}

// ExtractSectorEvents transforms sectorChanges, preCommitChanges, and sectorStateChanges to a MinerSectorEventList.
func ExtractSectorEvents(extState extraction.State, sectorChanges *miner.SectorChanges, preCommitChanges *miner.PreCommitChanges, sectorStateChanges *SectorStateEvents) (minermodel.MinerSectorEventList, error) {
	sectorStateEvents, err := ExtractMinerSectorStateEvents(extState, sectorStateChanges)
	if err != nil {
		return nil, err
	}

	sectorEvents := ExtractMinerSectorEvents(extState, sectorChanges)

	preCommitEvents := ExtractMinerPreCommitEvents(extState, preCommitChanges)

	out := make(minermodel.MinerSectorEventList, 0, len(sectorEvents)+len(preCommitEvents)+len(sectorStateEvents))
	out = append(out, sectorEvents...)
	out = append(out, preCommitEvents...)
	out = append(out, sectorStateEvents...)

	return out, nil
}

// ExtractMinerSectorStateEvents transforms the removed, recovering, faulted, and recovered sectors from `events` to a
// MinerSectorEventList.
func ExtractMinerSectorStateEvents(extState extraction.State, events *SectorStateEvents) (minermodel.MinerSectorEventList, error) {
	out := minermodel.MinerSectorEventList{}

	// all sectors removed this epoch are considered terminated, this includes both early terminations and expirations.
	if err := events.Removed.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(extState.CurrentTipSet().Height()),
			MinerID:   extState.Address().String(),
			StateRoot: extState.CurrentTipSet().ParentState().String(),
			SectorID:  u,
			Event:     minermodel.SectorTerminated,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if err := events.Recovering.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(extState.CurrentTipSet().Height()),
			MinerID:   extState.Address().String(),
			StateRoot: extState.CurrentTipSet().ParentState().String(),
			SectorID:  u,
			Event:     minermodel.SectorRecovering,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if err := events.Faulted.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(extState.CurrentTipSet().Height()),
			MinerID:   extState.Address().String(),
			StateRoot: extState.CurrentTipSet().ParentState().String(),
			SectorID:  u,
			Event:     minermodel.SectorFaulted,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if err := events.Recovered.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(extState.CurrentTipSet().Height()),
			MinerID:   extState.Address().String(),
			StateRoot: extState.CurrentTipSet().ParentState().String(),
			SectorID:  u,
			Event:     minermodel.SectorRecovered,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	return out, nil
}

// ExtractMinerSectorEvents transforms the added, extended and snapped sectors from `sectors` to a MinerSectorEventList.
func ExtractMinerSectorEvents(extState extraction.State, sectors *miner.SectorChanges) minermodel.MinerSectorEventList {
	out := make(minermodel.MinerSectorEventList, 0, len(sectors.Added)+len(sectors.Extended)+len(sectors.Snapped))

	// track sector add and commit-capacity add
	for _, add := range sectors.Added {
		event := minermodel.SectorAdded
		if len(add.DealIDs) == 0 {
			event = minermodel.CommitCapacityAdded
		}
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(extState.CurrentTipSet().Height()),
			MinerID:   extState.Address().String(),
			StateRoot: extState.CurrentTipSet().ParentState().String(),
			SectorID:  uint64(add.SectorNumber),
			Event:     event,
		})
	}

	// sector extension events
	for _, mod := range sectors.Extended {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(extState.CurrentTipSet().Height()),
			MinerID:   extState.Address().String(),
			StateRoot: extState.CurrentTipSet().ParentState().String(),
			SectorID:  uint64(mod.To.SectorNumber),
			Event:     minermodel.SectorExtended,
		})
	}

	// sector snapped events
	for _, snap := range sectors.Snapped {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(extState.CurrentTipSet().Height()),
			MinerID:   extState.Address().String(),
			StateRoot: extState.CurrentTipSet().ParentState().String(),
			SectorID:  uint64(snap.To.SectorNumber),
			Event:     minermodel.SectorSnapped,
		})
	}

	return out
}

// ExtractMinerPreCommitEvents transforms the added PreCommits from `preCommits` to a MinerSectorEventList.
func ExtractMinerPreCommitEvents(extState extraction.State, preCommits *miner.PreCommitChanges) minermodel.MinerSectorEventList {
	out := make(minermodel.MinerSectorEventList, len(preCommits.Added))
	// track precommit addition
	for i, add := range preCommits.Added {
		out[i] = &minermodel.MinerSectorEvent{
			Height:    int64(extState.CurrentTipSet().Height()),
			MinerID:   extState.Address().String(),
			StateRoot: extState.CurrentTipSet().ParentState().String(),
			SectorID:  uint64(add.Info.SectorNumber),
			Event:     minermodel.PreCommitAdded,
		}
	}

	return out
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
	_, span := otel.Tracer("").Start(ctx, "LoadSectorState")
	defer span.End()

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

// SectorStateEvents contains bitfields for sectors that were removed, recovered, faulted, and recovering.
type SectorStateEvents struct {
	// Removed sectors this epoch
	Removed bitfield.BitField
	// Recovering sectors this epoch
	Recovering bitfield.BitField
	// Faulted sectors this epoch
	Faulted bitfield.BitField
	// Recovered sectors this epoch
	Recovered bitfield.BitField
}

// DiffMinerSectorStates loads the SectorStates for the current and parent miner states in parallel from `extState`.
// Then compares current and parent SectorStates to produce a SectorStateEvents structure containing all sectors that are
// removed, recovering, faulted, and recovered for the state transition from parent miner state to current miner state.
func DiffMinerSectorStates(ctx context.Context, extState extraction.State) (*SectorStateEvents, error) {
	ctx, span := otel.Tracer("").Start(ctx, "DiffMinerSectorStates")
	defer span.End()
	var (
		previous, current *SectorStates
		err               error
	)

	// load previous and current miner sector states in parallel
	grp, grpCtx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		previous, err = LoadSectorState(grpCtx, extState.ParentState())
		if err != nil {
			return fmt.Errorf("loading previous sector states %w", err)
		}
		return nil
	})
	grp.Go(func() error {
		current, err = LoadSectorState(grpCtx, extState.CurrentState())
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

	return &SectorStateEvents{
		Removed:    removed,
		Recovering: recovering,
		Faulted:    faulted,
		Recovered:  recovered,
	}, nil

}
