package multisig

import (
	"context"

	"github.com/go-pg/pg/v10"
)

type MultisigTaskResult struct {
	TransactionModel MultisigTransactionList
}

func (m *MultisigTaskResult) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if len(m.TransactionModel) > 0 {
			return m.TransactionModel.PersistWithTx(ctx, tx)
		}
		return nil
	})
}

func (m *MultisigTaskResult) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(m.TransactionModel) > 0 {
		return m.TransactionModel.PersistWithTx(ctx, tx)
	}
	return nil
}

type MultisigTaskResultList []*MultisigTaskResult

func (ml MultisigTaskResultList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, res := range ml {
		if err := res.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
