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

	preCommitChanges, err := node.DiffPreCommits(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	sectorChanges, err := node.DiffSectors(ctx, a.Address, a.Current, a.Executed, ec.PrevState, ec.CurrState)
	if err != nil {
		return nil, err
	}

	sectorEventModel, err := extractMinerSectorEvents(ctx, node, a, ec, sectorChanges, preCommitChanges)
	if err != nil {
		return nil, err
	}

	return sectorEventModel, nil
}

func extractMinerSectorEvents(ctx context.Context, node actorstate.ActorStateAPI, a actorstate.ActorInfo, ec *MinerStateExtractionContext, sc *miner.SectorChanges, pc *miner.PreCommitChanges) (minermodel.MinerSectorEventList, error) {
	ctx, span := otel.Tracer("").Start(ctx, "extractMinerSectorEvents")
	defer span.End()

	ps, err := extractMinerPartitionsDiff(ctx, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner partition diff: %w", err)
	}

	out := minermodel.MinerSectorEventList{}
	sectorAdds := make(map[abi.SectorNumber]miner.SectorOnChainInfo)

	// if there were changes made to the miners partition lists
	if ps != nil {
		// build an index of removed sector expiration's for comparison below.

		removedSectors, err := ec.CurrState.LoadSectors(&ps.Removed)
		if err != nil {
			return nil, xerrors.Errorf("fetching miners removed sectors: %w", err)
		}
		rmExpireIndex := make(map[uint64]abi.ChainEpoch)
		for _, rm := range removedSectors {
			rmExpireIndex[uint64(rm.SectorNumber)] = rm.Expiration
		}

		// track terminated and expired sectors
		if err := ps.Removed.ForEach(func(u uint64) error {
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
		if err := ps.Recovering.ForEach(func(u uint64) error {
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
		if err := ps.Faulted.ForEach(func(u uint64) error {
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
		if err := ps.Recovered.ForEach(func(u uint64) error {
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
	}

	// if there were changes made to the miners sectors list
	if sc != nil {
		// track sector add and commit-capacity add
		for _, add := range sc.Added {
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
			sectorAdds[add.SectorNumber] = add
		}

		// track sector extensions
		for _, mod := range sc.Extended {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Current.Height()),
				MinerID:   a.Address.String(),
				StateRoot: a.Current.ParentState().String(),
				SectorID:  uint64(mod.To.SectorNumber),
				Event:     minermodel.SectorExtended,
			})
		}

	}

	// if there were changes made to the miners precommit list
	if pc != nil {
		// track precommit addition
		for _, add := range pc.Added {
			out = append(out, &minermodel.MinerSectorEvent{
				Height:    int64(a.Current.Height()),
				MinerID:   a.Address.String(),
				StateRoot: a.Current.ParentState().String(),
				SectorID:  uint64(add.Info.SectorNumber),
				Event:     minermodel.PreCommitAdded,
			})
		}
	}

	return out, nil
}

// PartitionStatus contains bitfileds of sectorID's that are removed, faulted, recovered and recovering.
type PartitionStatus struct {
	Removed    bitfield.BitField
	Faulted    bitfield.BitField
	Recovering bitfield.BitField
	Recovered  bitfield.BitField
}

func extractMinerPartitionsDiff(ctx context.Context, ec *MinerStateExtractionContext) (*PartitionStatus, error) {
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
	return &PartitionStatus{
		Removed:    removed,
		Faulted:    faulted,
		Recovering: recovering,
		Recovered:  recovered,
	}, nil
}
