package multisig

import (
	"context"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
	"go.opencensus.io/tag"
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

func (m *MultisigTransaction) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "multisig_transactions"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type MultisigTransactionList []*MultisigTransaction

func (ml MultisigTransactionList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "multisig_transactions"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}
