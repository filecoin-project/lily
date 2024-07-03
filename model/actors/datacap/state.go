package datacap

import (
	"context"

	"go.opencensus.io/tag"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

const (
	Added          = "ADDED"
	Removed        = "REMOVED"
	Modified       = "MODIFIED"
	Verifier       = "verifier"
	VerifierClient = "verifier_client"
)

type DataCapBalance struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	StateRoot string `pg:",pk,notnull"`
	Address   string `pg:",pk,notnull"`

	Event       string `pg:",notnull,type:data_cap_balance_event_type"`
	DataCap     string `pg:",notnull,type:numeric"`
	AddressType string `pg:",notnull"`
}

func (d *DataCapBalance) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "data_cap_balances"))

	return s.PersistModel(ctx, d)
}

type DataCapBalanceList []*DataCapBalance

func (d DataCapBalanceList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "data_cap_balances"))

	if len(d) == 0 {
		return nil
	}

	return s.PersistModel(ctx, d)
}
