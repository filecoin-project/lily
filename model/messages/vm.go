package messages

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type VMMessage struct {
	tableName struct{} `pg:"vm_messages"` // nolint: structcheck

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// StateRoot message was applied to.
	StateRoot string `pg:",pk,notnull"`
	// Cid of the message.
	Cid string `pg:",pk,notnull"`
	// On-chain message triggering the message.
	Source string `pg:",pk,notnull"`

	// From sender of message.
	From string `pg:",notnull"`
	// To receiver of message.
	To string `pg:",notnull"`
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
	// Params contained in message.
	Params string `pg:",type:jsonb"`
	// Return value of message.
	Returns string `pg:",type:jsonb"`
	// Root of AMT.
	// Absent when no events have been emitted.
	// See https://github.com/filecoin-project/FIPs/blob/master/FIPS/fip-0049.md#chain-commitment
	// If EthRPC is enabled, then the AMT is saved into the blockstore.
	EventsRoot string
}

func (v *VMMessage) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "vm_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, v)
}

type VMMessageList []*VMMessage

func (vl VMMessageList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(vl) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "vm_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(vl))
	return s.PersistModel(ctx, vl)
}
