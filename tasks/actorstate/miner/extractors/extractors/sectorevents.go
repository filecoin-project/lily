package extractors

import (
	"context"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/miner"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner/extractors"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

func init() {
	extractors.Register(&MinerSectorEvent{}, ExtractMinerSectorEvents)
}

func ExtractMinerSectorEvents(ctx context.Context, ec *extractors.MinerStateExtractionContext) (model.Persistable, error) {
	sectorChanges := new(miner.SectorChanges)
	preCommitChanges := new(miner.PreCommitChanges)
	if !ec.HasPreviousState() {
		msectors, err := ec.CurrState.LoadSectors(nil)
		if err != nil {
			return nil, err
		}

		sectorChanges.Added = make([]miner.SectorOnChainInfo, len(msectors))
		for idx, sector := range msectors {
			sectorChanges.Added[idx] = *sector
		}
	} else {
		var err error
		sectorChanges, err = extractors.GetSectorDiff(ctx, ec)
		if err != nil {
			return nil, xerrors.Errorf("diffing miner sectors: %w", err)
		}
		preCommitChanges, err = extractors.GetPreCommitDiff(ctx, ec)
		if err != nil {
			return nil, err
		}
	}
	return extractMinerSectorEvents(ctx, ec, sectorChanges, preCommitChanges)
}

func extractMinerSectorEvents(ctx context.Context, ec *extractors.MinerStateExtractionContext, sc *miner.SectorChanges, pc *miner.PreCommitChanges) (MinerSectorEventList, error) {
	ctx, span := global.Tracer("").Start(ctx, "extractMinerSectorEvents")
	defer span.End()

	ps, err := extractMinerPartitionsDiff(ctx, ec)
	if err != nil {
		return nil, xerrors.Errorf("extracting miner partition diff: %w", err)
	}

	out := MinerSectorEventList{}
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
			event := SectorTerminated
			expiration := rmExpireIndex[u]
			if expiration == ec.CurrTs.Height() {
				event = SectorExpired
			}
			out = append(out, &MinerSectorEvent{
				Height:    int64(ec.CurrTs.Height()),
				MinerID:   ec.Address.String(),
				StateRoot: ec.CurrTs.ParentState().String(),
				SectorID:  u,
				Event:     event,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners removed sectors: %w", err)
		}

		// track recovering sectors
		if err := ps.Recovering.ForEach(func(u uint64) error {
			out = append(out, &MinerSectorEvent{
				Height:    int64(ec.CurrTs.Height()),
				MinerID:   ec.Address.String(),
				StateRoot: ec.CurrTs.ParentState().String(),
				SectorID:  u,
				Event:     SectorRecovering,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners recovering sectors: %w", err)
		}

		// track faulted sectors
		if err := ps.Faulted.ForEach(func(u uint64) error {
			out = append(out, &MinerSectorEvent{
				Height:    int64(ec.CurrTs.Height()),
				MinerID:   ec.Address.String(),
				StateRoot: ec.CurrTs.ParentState().String(),
				SectorID:  u,
				Event:     SectorFaulted,
			})
			return nil
		}); err != nil {
			return nil, xerrors.Errorf("walking miners faulted sectors: %w", err)
		}

		// track recovered sectors
		if err := ps.Recovered.ForEach(func(u uint64) error {
			out = append(out, &MinerSectorEvent{
				Height:    int64(ec.CurrTs.Height()),
				MinerID:   ec.Address.String(),
				StateRoot: ec.CurrTs.ParentState().String(),
				SectorID:  u,
				Event:     SectorRecovered,
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
			event := SectorAdded
			if len(add.DealIDs) == 0 {
				event = CommitCapacityAdded
			}
			out = append(out, &MinerSectorEvent{
				Height:    int64(ec.CurrTs.Height()),
				MinerID:   ec.Address.String(),
				StateRoot: ec.CurrTs.ParentState().String(),
				SectorID:  uint64(add.SectorNumber),
				Event:     event,
			})
			sectorAdds[add.SectorNumber] = add
		}

		// track sector extensions
		for _, mod := range sc.Extended {
			out = append(out, &MinerSectorEvent{
				Height:    int64(ec.CurrTs.Height()),
				MinerID:   ec.Address.String(),
				StateRoot: ec.CurrTs.ParentState().String(),
				SectorID:  uint64(mod.To.SectorNumber),
				Event:     SectorExtended,
			})
		}

	}

	// if there were changes made to the miners precommit list
	if pc != nil {
		// track precommit addition
		for _, add := range pc.Added {
			out = append(out, &MinerSectorEvent{
				Height:    int64(ec.CurrTs.Height()),
				MinerID:   ec.Address.String(),
				StateRoot: ec.CurrTs.ParentState().String(),
				SectorID:  uint64(add.Info.SectorNumber),
				Event:     PreCommitAdded,
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

func extractMinerPartitionsDiff(ctx context.Context, ec *extractors.MinerStateExtractionContext) (*PartitionStatus, error) {
	_, span := global.Tracer("").Start(ctx, "extractMinerPartitionDiff") // nolint: ineffassign,staticcheck
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

const (
	PreCommitAdded   = "PRECOMMIT_ADDED"
	PreCommitExpired = "PRECOMMIT_EXPIRED"

	CommitCapacityAdded = "COMMIT_CAPACITY_ADDED"

	SectorAdded      = "SECTOR_ADDED"
	SectorExtended   = "SECTOR_EXTENDED"
	SectorFaulted    = "SECTOR_FAULTED"
	SectorRecovering = "SECTOR_RECOVERING"
	SectorRecovered  = "SECTOR_RECOVERED"

	SectorExpired    = "SECTOR_EXPIRED"
	SectorTerminated = "SECTOR_TERMINATED"
)

func init() {
}

type MinerSectorEvent struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	SectorID  uint64 `pg:",pk,use_zero"`
	StateRoot string `pg:",pk,notnull"`

	// https://github.com/go-pg/pg/issues/993
	// override the SQL type with enum type, see 1_chainwatch.go for enum definition
	//lint:ignore SA5008 duplicate tag allowed by go-pg
	Event string `pg:"type:miner_sector_event_type" pg:",pk,notnull"`
}

func (mse *MinerSectorEvent) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_events"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, mse)
}

type MinerSectorEventList []*MinerSectorEvent

func (l MinerSectorEventList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorEventList.Persist", trace.WithAttributes(label.Int("count", len(l))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_events"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(l) == 0 {
		return nil
	}

	return s.PersistModel(ctx, l)
}
