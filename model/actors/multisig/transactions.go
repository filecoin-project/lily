package multisig

import (
	"context"

	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"
)

type MultisigTransaction struct {
	MultisigID    string `pg:",pk,notnull"`
	StateRoot     string `pg:",pk,notnull"`
	Height        int64  `pg:",pk,notnull,use_zero"`
	TransactionID int64  `pg:",pk,notnull,use_zero"`

	// Transaction State
	To       string `pg:",notnull"`
	Value    string `pg:",notnull"`
	Method   uint64 `pg:",notnull,use_zero"`
	Params   []byte
	Approved []string `pg:",notnull"`
}

func (m *MultisigTransaction) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, m).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting multisig transaction: %w", err)
	}
	return nil
}

func (m *MultisigTransaction) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return m.PersistWithTx(ctx, tx)
	})
}

type MultisigTransactionList []*MultisigTransaction

func (ml MultisigTransactionList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, ml).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting multisig transaction list: %w", err)
	}
	return nil
}

func (ml MultisigTransactionList) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return ml.PersistWithTx(ctx, tx)
	})
}
