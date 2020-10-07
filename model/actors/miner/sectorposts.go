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
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,notnull,use_zero"`
	Epoch    int64  `pg:",pk,notnull"`

	PostMessageCID string
}

type MinerSectorPosts []MinerSectorPost

func (msp *MinerSectorPost) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, msp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner sector window post: %w", err)
	}
	return nil
}

// TODO: what makes this.
func NewMinerSectorPost() MinerSectorPosts {
	out := make([]MinerSectorPost, 0)

	return out
}

func (msps MinerSectorPosts) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return msps.PersistWithTx(ctx, tx)
	})
}

func (msps MinerSectorPosts) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorPosts.PersistWithTx", trace.WithAttributes(label.Int("count", len(msis))))
	defer span.End()
	for _, msp := range msps {
		if err := msp.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
