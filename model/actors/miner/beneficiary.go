package miner

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/metrics"
	"github.com/filecoin-project/lily/model"
)

type MinerBeneficiary struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	StateRoot string `pg:",pk,notnull"`
	MinerID   string `pg:",pk,notnull"`

	Beneficiary string `pg:",notnull"`

	Quota      string `pg:"type:numeric,notnull"`
	UsedQuota  string `pg:"type:numeric,notnull"`
	Expiration int64  `pg:",notnull,use_zero"`

	NewBeneficiary        string
	NewQuota              string `pg:"type:numeric"`
	NewExpiration         int64  `pg:",use_zero"`
	ApprovedByBeneficiary bool
	ApprovedByNominee     bool
}

func (m *MinerBeneficiary) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerBeneficiaryModel.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_beneficiaries"))
	metrics.RecordCount(ctx, metrics.PersistModel, 1)
	return s.PersistModel(ctx, m)
}

type MinerBeneficiaryList []*MinerBeneficiary

func (ml MinerBeneficiaryList) Persist(ctx context.Context, s model.StorageBatch, _ model.Version) error {
	ctx, span := otel.Tracer("").Start(ctx, "MinerBeneficiaryList.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_beneficiaries"))

	if len(ml) == 0 {
		return nil
	}
	metrics.RecordCount(ctx, metrics.PersistModel, len(ml))
	return s.PersistModel(ctx, ml)
}
