package blocks

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

func NewDrandBlockEntries(header *types.BlockHeader) DrandBlockEntries {
	var out DrandBlockEntries
	for _, ent := range header.BeaconEntries {
		out = append(out, &DrandBlockEntrie{
			Round: ent.Round,
			Block: header.Cid().String(),
		})
	}
	return out
}

// DrandBlockEntrie contains Drand randomness round numbers used in each block.
type DrandBlockEntrie struct {
	// Round is the round number of randomness used.
	Round uint64 `pg:",pk,use_zero"`
	// Block is the CID of the block.
	Block string `pg:",pk,notnull"`
}

func (dbe *DrandBlockEntrie) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "drand_block_entries"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, dbe)
}

type DrandBlockEntries []*DrandBlockEntrie

func (dbes DrandBlockEntries) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	if len(dbes) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "DrandBlockEntries.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(dbes)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "drand_block_entries"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(dbes))
	return s.PersistModel(ctx, dbes)
}
