package messages

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/opentracing/opentracing-go"
)

type BlockMessage struct {
	Block   string `pg:",pk,notnull"`
	Message string `pg:",pk,notnull"`
}

func (bm *BlockMessage) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, bm).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting block message: %w", err)
	}
	return nil
}

type BlockMessages []*BlockMessage

func (bms BlockMessages) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "BlockMessages.PersistWithTx", opentracing.Tags{"count": len(bms)})
	defer span.Finish()
	for _, bm := range bms {
		if err := bm.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
