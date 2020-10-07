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

type DrandEntrie struct {
	Round uint64 `pg:",pk,use_zero"`
	Data  []byte `pg:",notnull"`
}

func (de *DrandEntrie) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, de).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting drand entries: %w", err)
	}
	return nil
}

func NewDrandEnties(header *types.BlockHeader) DrandEntries {
	var out DrandEntries
	for _, ent := range header.BeaconEntries {
		out = append(out, &DrandEntrie{
			Round: ent.Round,
			Data:  ent.Data,
		})
	}
	return out
}

type DrandEntries []*DrandEntrie

func (des DrandEntries) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return des.PersistWithTx(ctx, tx)
	})
}

func (des DrandEntries) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "DrandEntries.PersistWithTx", trace.WithAttributes(label.Int("count", len(des))))
	defer span.End()
	if _, err := tx.ModelContext(ctx, &des).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting drand entries: %w", err)
	}
	return nil
}

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
	ctx, span := global.Tracer("").Start(ctx, "DrandBlockEntries.PersistWithTx", trace.WithAttributes(label.Int("count", len(dbes))))
	defer span.End()
	if _, err := tx.ModelContext(ctx, &dbes).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting drand block entries: %w", err)
	}
	return nil
}
