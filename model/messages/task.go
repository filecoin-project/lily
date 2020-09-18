package messages

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/opentracing/opentracing-go"
)

type MessageTaskResult struct {
	Messages      Messages
	BlockMessages BlockMessages
	Receipts      Receipts
}

func (mtr *MessageTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "MessageTaskResult.Persist")
	defer span.Finish()
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if err := mtr.Messages.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		if err := mtr.BlockMessages.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		if err := mtr.Receipts.PersistWithTx(ctx, tx); err != nil {
			return err
		}
		return nil
	})
}
