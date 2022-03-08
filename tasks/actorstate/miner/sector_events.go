package miner

import (
	"context"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type SectorEventsExtractor struct{}

func (SectorEventsExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "SectorEventsExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "SectorEventsExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("creating miner state extraction context: %w", err)
	}

	var sectorChanges *miner.SectorChanges
	var preCommitChanges *miner.PreCommitChanges
	if !ec.HasPreviousState() {
		// If the miner doesn't have previous state list all of its current sectors and precommits
		sectors, err := ec.CurrState.LoadSectors(nil)
		if err != nil {
			return nil, xerrors.Errorf("loading miner sectors: %w", err)
		}

		sectorChanges = miner.MakeSectorChanges()
		for i, sector := range sectors {
			sectorChanges.Added[i] = *sector
		}

		preCommitChanges = miner.MakePreCommitChanges()
		if err = ec.CurrState.ForEachPrecommittedSector(func(info miner.SectorPreCommitOnChainInfo) error {
			preCommitChanges.Added = append(preCommitChanges.Added, info)
			return nil
		}); err != nil {
			return nil, err
		}

	} else {
		// If the miner has previous state compute the list of new sectors and precommit in its current state.
		preCommitChanges, err = node.DiffPreCommits(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}

		sectorChanges, err = node.DiffSectors(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
		if err != nil {
			return nil, err
		}
	}

	sectorEventModel, err := extractSectorEvents(ctx, a, ec, sectorChanges, preCommitChanges)
	if err != nil {
		return nil, err
	}

	return sectorEventModel, nil
}

func extractSectorEvents(ctx context.Context, a actorstate.ActorInfo, ec *MinerStateExtractionContext, sc *miner.SectorChanges, pc *miner.PreCommitChanges) (minermodel.MinerSectorEventList, error) {
	ctx, span := otel.Tracer("").Start(ctx, "extractMinerSectorEvents")
	defer span.End()

	partitionEvents, err := extractMinerPartitionsDiff(ctx, a, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner partition diff: %w", err)
	}

	sectorEvents := extractMinerSectorEvents(a, sc)

	preCommitEvents := extractMinerPreCommitEvents(a, pc)

	out := make(minermodel.MinerSectorEventList, 0, len(partitionEvents)+len(sectorEvents)+len(preCommitEvents))
	out = append(out, partitionEvents...)
	out = append(out, sectorEvents...)
	out = append(out, preCommitEvents...)

	return out, nil
}

func extractMinerSectorEvents(a actorstate.ActorInfo, sectors *miner.SectorChanges) minermodel.MinerSectorEventList {
	out := make(minermodel.MinerSectorEventList, 0, len(sectors.Added)+len(sectors.Extended)+len(sectors.Snapped))

	// track sector add and commit-capacity add
	for _, add := range sectors.Added {
		event := minermodel.SectorAdded
		if len(add.DealIDs) == 0 {
			event = minermodel.CommitCapacityAdded
		}
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(a.Current.Height()),
			MinerID:   a.Address.String(),
			StateRoot: a.Current.ParentState().String(),
			SectorID:  uint64(add.SectorNumber),
			Event:     event,
		})
	}

	// sector extension events
	for _, mod := range sectors.Extended {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(a.Current.Height()),
			MinerID:   a.Address.String(),
			StateRoot: a.Current.ParentState().String(),
			SectorID:  uint64(mod.To.SectorNumber),
			Event:     minermodel.SectorExtended,
		})
	}

	// sector snapped events
	for _, snap := range sectors.Snapped {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(a.Current.Height()),
			MinerID:   a.Address.String(),
			StateRoot: a.Current.ParentState().String(),
			SectorID:  uint64(snap.To.SectorNumber),
			Event:     minermodel.SectorSnapped,
		})
	}

	return out
}

func extractMinerPreCommitEvents(a actorstate.ActorInfo, preCommits *miner.PreCommitChanges) minermodel.MinerSectorEventList {
	out := make(minermodel.MinerSectorEventList, len(preCommits.Added))
	// track precommit addition
	for i, add := range preCommits.Added {
		out[i] = &minermodel.MinerSectorEvent{
			Height:    int64(a.Current.Height()),
			MinerID:   a.Address.String(),
			StateRoot: a.Current.ParentState().String(),
			SectorID:  uint64(add.Info.SectorNumber),
			Event:     minermodel.PreCommitAdded,
		}
	}

	return out
}

func extractMinerPartitionsDiff(ctx context.Context, a actorstate.ActorInfo, ec *MinerStateExtractionContext) (minermodel.MinerSectorEventList, error) {
	_, span := otel.Tracer("").Start(ctx, "extractMinerPartitionDiff") // nolint: ineffassign,staticcheck
	defer span.End()

	// short circuit genesis state.
	if !ec.HasPreviousState() {
		return nil, nil
	}

	dlDiff, err := miner.DiffDeadlines(ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	if dlDiff == nil {
		return nil, nil
	}

	removed := bitfield.New()
	faulted := bitfield.New()
	recovered := bitfield.New()
	recovering := bitfield.New()

	for _, deadline := range dlDiff {
		for _, partition := range deadline {
			removed, err = bitfield.MergeBitFields(removed, partition.Removed)
			if err != nil {
				return nil, err
			}
			faulted, err = bitfield.MergeBitFields(faulted, partition.Faulted)
			if err != nil {
				return nil, err
			}
			recovered, err = bitfield.MergeBitFields(recovered, partition.Recovered)
			if err != nil {
				return nil, err
			}
			recovering, err = bitfield.MergeBitFields(recovering, partition.Recovering)
			if err != nil {
				return nil, err
			}
		}
	}
	// build an index of removed sector expiration's for comparison below.

	removedSectors, err := ec.CurrState.LoadSectors(&removed)
	if err != nil {
		return nil, xerrors.Errorf("fetching miners removed sectors: %w", err)
	}
	rmExpireIndex := make(map[uint64]abi.ChainEpoch)
	for _, rm := range removedSectors {
		rmExpireIndex[uint64(rm.SectorNumber)] = rm.Expiration
	}

	out := minermodel.MinerSectorEventList{}

	// track terminated and expired sectors
	if err := removed.ForEach(func(u uint64) error {
		event := minermodel.SectorTerminated
		expiration := rmExpireIndex[u]
		if expiration == a.Current.Height() {
			event = minermodel.SectorExpired
		}
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(a.Current.Height()),
			MinerID:   a.Address.String(),
			StateRoot: a.Current.ParentState().String(),
			SectorID:  u,
			Event:     event,
		})
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("walking miners removed sectors: %w", err)
	}

	// track recovering sectors
	if err := recovering.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(a.Current.Height()),
			MinerID:   a.Address.String(),
			StateRoot: a.Current.ParentState().String(),
			SectorID:  u,
			Event:     minermodel.SectorRecovering,
		})
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("walking miners recovering sectors: %w", err)
	}

	// track faulted sectors
	if err := faulted.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(a.Current.Height()),
			MinerID:   a.Address.String(),
			StateRoot: a.Current.ParentState().String(),
			SectorID:  u,
			Event:     minermodel.SectorFaulted,
		})
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("walking miners faulted sectors: %w", err)
	}

	// track recovered sectors
	if err := recovered.ForEach(func(u uint64) error {
		out = append(out, &minermodel.MinerSectorEvent{
			Height:    int64(a.Current.Height()),
			MinerID:   a.Address.String(),
			StateRoot: a.Current.ParentState().String(),
			SectorID:  u,
			Event:     minermodel.SectorRecovered,
		})
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("walking miners recovered sectors: %w", err)
	}
	return out, nil
}
