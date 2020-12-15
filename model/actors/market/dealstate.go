package market

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MarketDealState struct {
	Height           int64  `pg:",pk,notnull,use_zero"`
	DealID           uint64 `pg:",pk,use_zero"`
	SectorStartEpoch int64  `pg:",pk,use_zero"`
	LastUpdateEpoch  int64  `pg:",pk,use_zero"`
	SlashEpoch       int64  `pg:",pk,use_zero"`

	StateRoot string `pg:",notnull"`
}

func (ds *MarketDealState) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, ds)
}

type MarketDealStates []*MarketDealState

func (dss MarketDealStates) Persist(ctx context.Context, s model.StorageBatch) error {
	ctx, span := global.Tracer("").Start(ctx, "MarketDealStates.PersistWithTx", trace.WithAttributes(label.Int("count", len(dss))))
	defer span.End()
	for _, ds := range dss {
		if err := s.PersistModel(ctx, ds); err != nil {
			return err
		}
	}
	return nil
}
