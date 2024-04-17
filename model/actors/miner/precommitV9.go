package miner

import (
	"context"
	"fmt"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

//revive:disable
type MinerPreCommitInfoV9 struct {
	tableName struct{} `pg:"miner_pre_commit_infos_v9"` // nolint: structcheck

	Height    int64  `pg:",pk,notnull,use_zero"`
	StateRoot string `pg:",pk,notnull"`
	MinerID   string `pg:",pk,notnull"`
	SectorID  uint64 `pg:",pk,use_zero"`

	PreCommitDeposit string `pg:"type:numeric,notnull"`
	PreCommitEpoch   int64  `pg:",use_zero"`

	SealedCID       string   `pg:",notnull"`
	SealRandEpoch   int64    `pg:",use_zero"`
	ExpirationEpoch int64    `pg:",use_zero"`
	DealIDS         []uint64 `pg:",array"`
	UnsealedCID     string   `pg:",notnull"`
}

func (mpi *MinerPreCommitInfoV9) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		return nil, false
	case 1:
		return mpi, true
	default:
		return nil, false
	}
}

func (mpi *MinerPreCommitInfoV9) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_pre_commit_infos"))

	m, ok := mpi.AsVersion(version)
	if !ok {
		return fmt.Errorf("MinerPreCommitInfoV9 not supported for schema version %s", version)
	}

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type MinerPreCommitInfoV9List []*MinerPreCommitInfoV9

func (ml MinerPreCommitInfoV9List) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerPreCommitInfoV8List.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(ml)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_pre_commit_infos"))

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
