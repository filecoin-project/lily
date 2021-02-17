package msapprovals

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

type MultisigApproval struct {
	tableName      struct{} `pg:"multisig_approvals"` // nolint: structcheck,unused
	Height         int64    `pg:",pk,notnull,use_zero"`
	StateRoot      string   `pg:",pk,notnull"`
	MultisigID     string   `pg:",pk,notnull"`
	Message        string   `pg:",pk,notnull"` // cid of message
	TransactionID  int64    `pg:",notnull,use_zero"`
	Method         uint64   `pg:",notnull,use_zero"` // method number used for the approval 2=propose, 3=approve
	Threshold      uint64   `pg:",notnull,use_zero"`
	InitialBalance string   `pg:"type:numeric,notnull"`
}

func (ma *MultisigApproval) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "multisig_approvals"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, ma)
}

type MultisigApprovalList []*MultisigApproval

func (mal MultisigApprovalList) Persist(ctx context.Context, s model.StorageBatch) error {
	if len(mal) == 0 {
		return nil
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "multisig_approvals"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, mal)
}
