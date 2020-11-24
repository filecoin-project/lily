package blocks

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

type BlockHeader struct {
	Height          int64  `pg:",pk,use_zero,notnull"`
	Cid             string `pg:",pk,notnull"`
	Miner           string `pg:",notnull"`
	ParentWeight    string `pg:",notnull"`
	ParentBaseFee   string `pg:",notnull"`
	ParentStateRoot string `pg:",notnull"`

	WinCount      int64  `pg:",use_zero"`
	Timestamp     uint64 `pg:",use_zero"`
	ForkSignaling uint64 `pg:",use_zero"`
}

func NewBlockHeader(bh *types.BlockHeader) *BlockHeader {
	return &BlockHeader{
		Cid:             bh.Cid().String(),
		Miner:           bh.Miner.String(),
		ParentWeight:    bh.ParentWeight.String(),
		ParentBaseFee:   bh.ParentBaseFee.String(),
		ParentStateRoot: bh.ParentStateRoot.String(),
		Height:          int64(bh.Height),
		WinCount:        bh.ElectionProof.WinCount,
		Timestamp:       bh.Timestamp,
		ForkSignaling:   bh.ForkSignaling,
	}
}

func (bh *BlockHeader) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, bh).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting block header: %w", err)
	}
	return nil
}

type BlockHeaders []*BlockHeader

func (bh BlockHeaders) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return bh.PersistWithTx(ctx, tx)
	})
}

func (bh BlockHeaders) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(bh) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "BlockHeaders.PersistWithTx", trace.WithAttributes(label.Int("count", len(bh))))
	defer span.End()
	if _, err := tx.ModelContext(ctx, &bh).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting block headers: %w", err)
	}
	return nil
}
