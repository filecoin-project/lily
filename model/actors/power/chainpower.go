package power

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

type ChainPower struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	StateRoot string `pg:",pk"`

	TotalRawBytesPower string `pg:",notnull"`
	TotalQABytesPower  string `pg:",notnull"`

	TotalRawBytesCommitted string `pg:",notnull"`
	TotalQABytesCommitted  string `pg:",notnull"`

	TotalPledgeCollateral string `pg:",notnull"`

	QASmoothedPositionEstimate string `pg:",notnull"`
	QASmoothedVelocityEstimate string `pg:",notnull"`

	MinerCount              uint64 `pg:",use_zero"`
	ParticipatingMinerCount uint64 `pg:",use_zero"`
}

func (cp *ChainPower) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainPower.PersistWithTx")
	defer span.End()
	return s.PersistModel(ctx, cp)
}

// ChainPowerList is a slice of ChainPowers for batch insertion.
type ChainPowerList []*ChainPower

// PersistWithTx makes a batch insertion of the list using the given
// transaction.
func (cpl ChainPowerList) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainPowerList.PersistWithTx", trace.WithAttributes(label.Int("count", len(cpl))))
	defer span.End()

	if len(cpl) == 0 {
		return nil
	}
	return s.PersistModel(ctx, cpl)
}
