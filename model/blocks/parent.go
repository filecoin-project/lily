package blocks

import (
	"context"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lotus/chain/types"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type BlockParent struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"block_parents"`

	Height int64  `pg:",pk,notnull,use_zero"`
	Block  string `pg:",pk,notnull"`
	Parent string `pg:",notnull"`
}

func (bp *BlockParent) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "block_parents"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, bp)
}

type BlockParents []*BlockParent

func NewBlockParents(header *types.BlockHeader) BlockParents {
	var out BlockParents
	for _, p := range header.Parents {
		out = append(out, &BlockParent{
			Height: int64(header.Height),
			Block:  header.Cid().String(),
			Parent: p.String(),
		})
	}
	return out
}

func (bps BlockParents) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(bps) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "BlockParents.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(bps)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "block_parents"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(bps))
	return s.PersistModel(ctx, bps)
}
