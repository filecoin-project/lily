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

type DrandBlockEntrie struct {
	Round uint64 `pg:",pk,use_zero"`
	Block string `pg:",notnull"`
}

func (dbe *DrandBlockEntrie) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, dbe).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting drand block entries: %w", err)
	}
	return nil
}

type DrandBlockEntries []*DrandBlockEntrie

func (dbes DrandBlockEntries) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return dbes.PersistWithTx(ctx, tx)
	})
}

func (dbes DrandBlockEntries) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(dbes) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "DrandBlockEntries.PersistWithTx", trace.WithAttributes(label.Int("count", len(dbes))))
	defer span.End()
	if _, err := tx.ModelContext(ctx, &dbes).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting drand block entries: %w", err)
	}
	return nil
}
