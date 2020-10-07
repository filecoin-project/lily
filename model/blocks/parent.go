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

type BlockParent struct {
	Block  string `pg:",pk,notnull"`
	Parent string `pg:",notnull"`
}

func (bp *BlockParent) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, bp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting block parents: %w", err)
	}
	return nil
}

type BlockParents []*BlockParent

func NewBlockParents(header *types.BlockHeader) BlockParents {
	var out BlockParents
	for _, p := range header.Parents {
		out = append(out, &BlockParent{
			Block:  header.Cid().String(),
			Parent: p.String(),
		})
	}
	return out
}

func (bps BlockParents) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return bps.PersistWithTx(ctx, tx)
	})
}

func (bps BlockParents) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "BlockParents.PersistWithTx", trace.WithAttributes(label.Int("count", len(bps))))
	defer span.End()
	if _, err := tx.ModelContext(ctx, &bps).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting block parents: %w", err)
	}
	return nil
}
