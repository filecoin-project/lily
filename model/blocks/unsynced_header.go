package blocks

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"

	"github.com/filecoin-project/lotus/chain/types"
)

type UnsyncedBlockHeader struct {
	Height          int64  `pg:",pk,use_zero,notnull"`
	Cid             string `pg:",pk,notnull"`
	Miner           string `pg:",notnull"`
	ParentWeight    string `pg:",notnull"`
	ParentBaseFee   string `pg:",notnull"`
	ParentStateRoot string `pg:",notnull"`

	WinCount      int64  `pg:",use_zero"`
	Timestamp     uint64 `pg:",use_zero"`
	ForkSignaling uint64 `pg:",use_zero"`
	IsOrphan      bool   `pg:",notnull"`
}

func NewUnsyncedBlockHeader(bh *types.BlockHeader) *UnsyncedBlockHeader {
	return &UnsyncedBlockHeader{
		Cid:             bh.Cid().String(),
		Miner:           bh.Miner.String(),
		ParentWeight:    bh.ParentWeight.String(),
		ParentBaseFee:   bh.ParentBaseFee.String(),
		ParentStateRoot: bh.ParentStateRoot.String(),
		Height:          int64(bh.Height),
		WinCount:        bh.ElectionProof.WinCount,
		Timestamp:       bh.Timestamp,
		ForkSignaling:   bh.ForkSignaling,
		IsOrphan:        false,
	}
}

func (bh *UnsyncedBlockHeader) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "unsynced_block_headers"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, bh)
}

type UnsyncedBlockHeaders []*UnsyncedBlockHeader

func (bhl UnsyncedBlockHeaders) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	if len(bhl) == 0 {
		return nil
	}
	ctx, span := otel.Tracer("").Start(ctx, "UnsyncedBlockHeaders.Persist")
	if span.IsRecording() {
		span.SetAttributes(attribute.Int("count", len(bhl)))
	}
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "block_headers"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(bhl))
	return s.PersistModel(ctx, bhl)
}
