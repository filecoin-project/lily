package miner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type MinerPreCommitInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	SectorID  uint64 `pg:",pk,use_zero"`
	StateRoot string `pg:",pk,notnull"`

	SealedCID       string `pg:",notnull"`
	SealRandEpoch   int64  `pg:",use_zero"`
	ExpirationEpoch int64  `pg:",use_zero"`

	PreCommitDeposit   string `pg:"type:numeric,notnull"`
	PreCommitEpoch     int64  `pg:",use_zero"`
	DealWeight         string `pg:"type:numeric,notnull"`
	VerifiedDealWeight string `pg:"type:numeric,notnull"`

	IsReplaceCapacity      bool
	ReplaceSectorDeadline  uint64 `pg:",use_zero"`
	ReplaceSectorPartition uint64 `pg:",use_zero"`
	ReplaceSectorNumber    uint64 `pg:",use_zero"`
}

type MinerPreCommitInfoV0 struct {
	tableName struct{} `pg:"miner_pre_commit_infos"` // nolint: structcheck
	Height    int64    `pg:",pk,notnull,use_zero"`
	MinerID   string   `pg:",pk,notnull"`
	SectorID  uint64   `pg:",pk,use_zero"`
	StateRoot string   `pg:",pk,notnull"`

	SealedCID       string `pg:",notnull"`
	SealRandEpoch   int64  `pg:",use_zero"`
	ExpirationEpoch int64  `pg:",use_zero"`

	PreCommitDeposit   string `pg:",notnull"`
	PreCommitEpoch     int64  `pg:",use_zero"`
	DealWeight         string `pg:",notnull"`
	VerifiedDealWeight string `pg:",notnull"`

	IsReplaceCapacity      bool
	ReplaceSectorDeadline  uint64 `pg:",use_zero"`
	ReplaceSectorPartition uint64 `pg:",use_zero"`
	ReplaceSectorNumber    uint64 `pg:",use_zero"`
}

func (mpi *MinerPreCommitInfo) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if mpi == nil {
			return (*MinerPreCommitInfoV0)(nil), true
		}

		return &MinerPreCommitInfoV0{
			Height:                 mpi.Height,
			MinerID:                mpi.MinerID,
			SectorID:               mpi.SectorID,
			StateRoot:              mpi.StateRoot,
			SealedCID:              mpi.SealedCID,
			SealRandEpoch:          mpi.SealRandEpoch,
			ExpirationEpoch:        mpi.ExpirationEpoch,
			PreCommitDeposit:       mpi.PreCommitDeposit,
			PreCommitEpoch:         mpi.PreCommitEpoch,
			DealWeight:             mpi.DealWeight,
			VerifiedDealWeight:     mpi.VerifiedDealWeight,
			IsReplaceCapacity:      mpi.IsReplaceCapacity,
			ReplaceSectorDeadline:  mpi.ReplaceSectorDeadline,
			ReplaceSectorPartition: mpi.ReplaceSectorPartition,
			ReplaceSectorNumber:    mpi.ReplaceSectorNumber,
		}, true
	case 1:
		return mpi, true
	default:
		return nil, false
	}
}

func (mpi *MinerPreCommitInfo) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_pre_commit_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	m, ok := mpi.AsVersion(version)
	if !ok {
		return fmt.Errorf("MinerPreCommitInfo not supported for schema version %s", version)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type MinerPreCommitInfoList []*MinerPreCommitInfo

func (ml MinerPreCommitInfoList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerPreCommitInfoList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(ml)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_pre_commit_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}

	if version.Major != 1 {
		// Support older versions, but in a non-optimal way
		for _, m := range ml {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}
