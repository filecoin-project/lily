package messages

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type VmMessage struct {
	tableName struct{} `pg:"vm_messages"`

	// Height message was executed at.
	Height int64 `pg:",pk,notnull,use_zero"`
	// TODO verify was this the result of the message or the sr it was applied to?
	// StateRoot created by messages execution.
	StateRoot string `pg:",pk,notnull"`
	// Cid of the message.
	Cid string `pg:",pk,notnull"`

	// Parent message triggering the message.
	Parent string
	// From sender of message.
	From string `pg:",notnull"`
	// To receiver of message.
	To string `pg:",notnull"`
	// Value FIL contained in message.
	Value string `pg:"type:numeric,notnull"`
	// Method called on To (receiver)
	Method string `pg:",use_zero"`
	// ActorName of To (receiver)
	ActorName string `pg:",notnull"`
	// ExitCode of message execution.
	ExitCode int64 `pg:",use_zero"`
	// GasUsed by message.
	GasUsed int64 `pg:",use_zero"`
	// Params contained in message.
	Params string `pg:",type:jsonb"`
	// Return value of message.
	Return string `pg:",type:jsonb"`
}

func (v *VmMessage) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "vm_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, v)
}

type VmMessageList []*VmMessage

func (vl VmMessageList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(vl) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "vm_messages"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	metrics.RecordCount(ctx, metrics.PersistModel, len(vl))
	return s.PersistModel(ctx, vl)
}
