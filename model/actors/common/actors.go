package common

import (
	"context"
	"github.com/go-pg/pg/v10"
)

type Actor struct {
	ID        string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`
	Code      string `pg:",notnull"`
	Head      string `pg:",notnull"`
	Balance   string `pg:",notnull"`
	Nonce     uint64 `pg:",use_zero"`
}

func (a *Actor) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, a).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil
}

type ActorState struct {
	Head  string `pg:",pk,notnull"`
	Code  string `pg:",pk,notnull"`
	State string `pg:",notnull"`
}

func (s *ActorState) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, s).
		OnConflict("do nothing").
		Insert(); err != nil {
		return err
	}
	return nil

}
