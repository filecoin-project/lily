package chain

import (
	"context"

	"github.com/go-pg/pg/v10"

	"github.com/filecoin-project/sentinel-visor/model"
)

type Storage interface {
	Persist(ctx context.Context, p model.PersistableWithTx) error
}

var _ Storage = (*NullStorage)(nil)

type NullStorage struct {
}

func (*NullStorage) Persist(ctx context.Context, p model.PersistableWithTx) error {
	log.Debugw("Not persisting data")
	return nil
}

type PersistableWithTxList []model.PersistableWithTx

var _ model.PersistableWithTx = (PersistableWithTxList)(nil)

func (pl PersistableWithTxList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	for _, p := range pl {
		if p == nil {
			continue
		}
		if err := p.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
