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
	// ETH Address of the sender.
	From string `pg:",notnull"`
	// ETH Address of the receiver.
	To string `pg:",notnull"`
	// Filecoin Address of the sender.
	FromFilecoinAddress string `pg:",notnull"`
	// Filecoin Address of the receiver.
	ToFilecoinAddress string `pg:",notnull"`

	// Value attoFIL contained in message.
	Value string `pg:"type:numeric,notnull"`
	// Method called on To (receiver).
	Method uint64 `pg:",notnull,use_zero"`
	// Method in readable name.
	ParsedMethod string `pg:",notnull"`
	// ActorCode of To (receiver).
	ActorCode string `pg:",notnull"`
	// ExitCode of message execution.
	ExitCode int64 `pg:",notnull,use_zero"`
	// Params contained in message encode in eth bytes.
	Params string `pg:",notnull"`
	// Returns value of message receipt encode in eth bytes.
	Returns string `pg:",notnull"`
	// Index indicating the order of the messages execution.
	Index uint64 `pg:",notnull,use_zero"`
	// Params contained in message.
	ParsedParams string `pg:",type:jsonb"`
	// Returns value of message receipt.
	ParsedReturns string `pg:",type:jsonb"`
	// Params codec.
	ParamsCodec uint64 `pg:",notnull,use_zero"`
	// Returns codec.
	ReturnsCodec uint64 `pg:",notnull,use_zero"`
}

func (f *FEVMTrace) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_traces"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, f)
}

type FEVMTraceList []*FEVMTrace

func (f FEVMTraceList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(f) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "fevm_traces"))
	metrics.RecordCount(ctx, metrics.PersistModel, len(f))
	return s.PersistModel(ctx, f)
}
