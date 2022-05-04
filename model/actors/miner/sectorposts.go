package miner

import (
	"context"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type MinerSectorPost struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"miner_sector_posts"`

	Height   int64  `pg:",pk,notnull,use_zero"`
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,notnull,use_zero"`

	PostMessageCID string
}

type MinerSectorPostList []*MinerSectorPost

func (msp *MinerSectorPost) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_posts"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, msp)
}

func (ml MinerSectorPostList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerSectorPostList.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(ml)))
	}
	defer span.End()
	if len(ml) == 0 {
		return nil
	}

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_sector_posts"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}
