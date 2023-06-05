package fevm

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type FEVMTrace struct {
	tableName struct{} `pg:"fevm_traces"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// StateRoot message was applied to.
	MessageStateRoot string `pg:",pk,notnull"`
	// On-chain message triggering the message.
	MessageCid string `pg:",pk,notnull"`
	// On-chain message ETH transaction hash
	TransactionHash string `pg:",notnull"`

	// Cid of the trace.
	TraceCid string `pg:",pk,notnull"`
	// Filecoin Address of the sender.
	From string `pg:",notnull"`
	// Filecoin Address of the receiver.
	To string `pg:",notnull"`
	// ETH Address of the sender.
	FromEthAddress string `pg:",notnull"`
	// ETH Address of the receiver.
	ToEthAddress string `pg:",notnull"`

	// Value attoFIL contained in message.
	Value string `pg:"type:numeric,notnull"`
	// Method called on To (receiver).
	Method uint64 `pg:",notnull,use_zero"`
	// ActorCode of To (receiver).
	ActorCode string `pg:",notnull"`
	// ExitCode of message execution.
	ExitCode int64 `pg:",notnull,use_zero"`
	// GasUsed by message.
	GasUsed int64 `pg:",notnull,use_zero"`
	// Params contained in message encode in base64.
	Params string `pg:",notnull"`
	// Returns value of message receipt encode in base64.
	Returns string `pg:",notnull"`
	// Index indicating the order of the messages execution.
	Index uint64 `pg:",notnull,use_zero"`
	// Params contained in message.
	ParsedParams string `pg:",type:jsonb"`
	// Returns value of message receipt.
	ParsedReturns string `pg:",type:jsonb"`
}

func (v *FEVMTrace) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_traces"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, v)
}

type FEVMTraceList []*FEVMTrace

func (vl FEVMTraceList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(vl) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_traces"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(vl))
	return s.PersistModel(ctx, vl)
}
