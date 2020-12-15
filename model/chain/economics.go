package chain

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

type ChainEconomics struct {
	tableName       struct{} `pg:"chain_economics"` // nolint: structcheck,unused
	ParentStateRoot string   `pg:",notnull"`
	CirculatingFil  string   `pg:",notnull"`
	VestedFil       string   `pg:",notnull"`
	MinedFil        string   `pg:",notnull"`
	BurntFil        string   `pg:",notnull"`
	LockedFil       string   `pg:",notnull"`
}

func (c *ChainEconomics) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, c)
}

type ChainEconomicsList []*ChainEconomics

func (l ChainEconomicsList) Persist(ctx context.Context, s model.StorageBatch) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ChainEconomicsList.Persist", trace.WithAttributes(label.Int("count", len(l))))
	defer span.End()
	return s.PersistModel(ctx, l)
}
