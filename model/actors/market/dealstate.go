package market

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

type MarketDealState struct {
	DealID           uint64 `pg:",pk,use_zero"`
	SectorStartEpoch int64  `pg:",pk,use_zero"`
	LastUpdateEpoch  int64  `pg:",pk,use_zero"`
	SlashEpoch       int64  `pg:",pk,use_zero"`

	StateRoot string `pg:",notnull"`
}

func (ds *MarketDealState) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, ds).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting marker deal state: %v", err)
	}
	return nil
}

type MarketDealStates []*MarketDealState

func (dss MarketDealStates) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MarketDealStates.PersistWithTx", trace.WithAttributes(label.Int("count", len(dss))))
	defer span.End()
	for _, ds := range dss {
		if err := ds.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
