package messages

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type ReceiptReturn struct {
	tableName struct{} `pg:"receipt_returns"` // nolint: structcheck
	Message   string   `pg:",pk,notnull"`
	Return    []byte
	// Result returned from executing a message parsed and serialized as a JSON object.
	ParsedReturn string `pg:",type:jsonb"`
}

func (m *ReceiptReturn) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "receipt_return"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type ReceiptReturnList []*ReceiptReturn

func (rl ReceiptReturnList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "receipt_return"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(rl))
	return s.PersistModel(ctx, rl)
}
