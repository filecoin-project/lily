package messages

import (
	"context"

	"github.com/go-pg/pg/v10"
)

type MessageTaskResult struct {
	Messages      Messages
	BlockMessages BlockMessages
	Receipts      Receipts
}

func (mtr *MessageTaskResult) Persist(ctx context.Context, db *pg.DB) error {
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
