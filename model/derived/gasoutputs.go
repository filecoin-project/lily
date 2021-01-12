package derived

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

type GasOutputs struct {
	tableName          struct{} `pg:"derived_gas_outputs"` //nolint: structcheck,unused
	Height             int64    `pg:",pk,use_zero,notnull"`
	Cid                string   `pg:",pk,notnull"`
	StateRoot          string   `pg:",pk,notnull"`
	From               string   `pg:",notnull"`
	To                 string   `pg:",notnull"`
	Value              string   `pg:"type:numeric,notnull"`
	GasFeeCap          string   `pg:"type:numeric,notnull"`
	GasPremium         string   `pg:"type:numeric,notnull"`
	GasLimit           int64    `pg:",use_zero,notnull"`
	SizeBytes          int      `pg:",use_zero,notnull"`
	Nonce              uint64   `pg:",use_zero,notnull"`
	Method             uint64   `pg:",use_zero,notnull"`
	ActorName          string   `pg:",notnull"`
	ExitCode           int64    `pg:",use_zero,notnull"`
	GasUsed            int64    `pg:",use_zero,notnull"`
	ParentBaseFee      string   `pg:"type:numeric,notnull"`
	BaseFeeBurn        string   `pg:"type:numeric,notnull"`
	OverEstimationBurn string   `pg:"type:numeric,notnull"`
	MinerPenalty       string   `pg:"type:numeric,notnull"`
	MinerTip           string   `pg:"type:numeric,notnull"`
	Refund             string   `pg:"type:numeric,notnull"`
	GasRefund          int64    `pg:",use_zero,notnull"`
	GasBurned          int64    `pg:",use_zero,notnull"`
}

func (g *GasOutputs) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, g)
}

type GasOutputsList []*GasOutputs

func (l GasOutputsList) Persist(ctx context.Context, s model.StorageBatch) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "GasOutputsList.Persist", trace.WithAttributes(label.Int("count", len(l))))
	defer span.End()
	return s.PersistModel(ctx, l)
}
