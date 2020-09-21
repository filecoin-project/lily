package init

import (
	"context"
	"github.com/go-pg/pg/v10"
)

type IdAddress struct {
	ID        string `pg:",pk,notnull"`
	Address   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`
}

func (ia *IdAddress) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, ia).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil
}

type IdAddressList []*IdAddress

func (ias IdAddressList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, ia := range ias {
		if err := ia.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func (ias IdAddressList) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return ias.PersistWithTx(ctx, tx)
	})
}
