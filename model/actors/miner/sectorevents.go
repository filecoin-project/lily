package miner

import (
	"context"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

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

	// specs-actors v7
	SectorSnapped = "SECTOR_SNAPPED"
)

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

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, mse)
}

type MinerSectorEventList []*MinerSectorEvent

func (l MinerSectorEventList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerSectorEventList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(l)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_events"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(l) == 0 {
		return nil
	}

	metrics.RecordCount(ctx, metrics.PersistModel, len(l))
	return s.PersistModel(ctx, l)
}
