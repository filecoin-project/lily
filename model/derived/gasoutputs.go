package derived

import (
	"context"

	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

type GasOutputs struct {
	tableName          struct{} `pg:"derived_gas_outputs"` //nolint: structcheck,unused
	Height             int64    `pg:",pk,use_zero,notnull"`
	Cid                string   `pg:",pk,notnull"`
	StateRoot          string   `pg:",pk,notnull"`
	From               string   `pg:",notnull"`
	To                 string   `pg:",notnull"`
	Value              string   `pg:",notnull"`
	GasFeeCap          string   `pg:",notnull"`
	GasPremium         string   `pg:",notnull"`
	GasLimit           int64    `pg:",use_zero,notnull"`
	SizeBytes          int      `pg:",use_zero,notnull"`
	Nonce              uint64   `pg:",use_zero,notnull"`
	Method             uint64   `pg:",use_zero,notnull"`
	ActorName          string   `pg:",notnull"`
	ExitCode           int64    `pg:",use_zero,notnull"`
	GasUsed            int64    `pg:",use_zero,notnull"`
	ParentBaseFee      string   `pg:",notnull"`
	BaseFeeBurn        string   `pg:",notnull"`
	OverEstimationBurn string   `pg:",notnull"`
	MinerPenalty       string   `pg:",notnull"`
	MinerTip           string   `pg:",notnull"`
	Refund             string   `pg:",notnull"`
	GasRefund          int64    `pg:",use_zero,notnull"`
	GasBurned          int64    `pg:",use_zero,notnull"`
}

func (g *GasOutputs) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "derived_gas_outputs"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, g)
}

type GasOutputsList []*GasOutputs

func (l GasOutputsList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "GasOutputsList.Persist", trace.WithAttributes(label.Int("count", len(l))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "derived_gas_outputs"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, l)
}
