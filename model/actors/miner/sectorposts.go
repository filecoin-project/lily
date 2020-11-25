package miner

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

type MinerSectorPost struct {
	Height   int64  `pg:",pk,notnull,use_zero"`
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,notnull,use_zero"`

	PostMessageCID string
}

type MinerSectorPostList []*MinerSectorPost

func (msp *MinerSectorPost) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, msp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner sector window post: %w", err)
	}
	return nil
}

func (ml MinerSectorPostList) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return ml.PersistWithTx(ctx, tx)
	})
}

func (ml MinerSectorPostList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorPostList.PersistWithTx", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &ml).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner sector post list: %w", err)
	}
	return nil
}
