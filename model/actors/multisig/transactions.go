package multisig

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
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

func (m *MultisigTransaction) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, m)
}

type MultisigTransactionList []*MultisigTransaction

func (ml MultisigTransactionList) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, ml)
}
