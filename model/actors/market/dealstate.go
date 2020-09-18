package market

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/opentracing/opentracing-go"
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "MarketDealStates.PersistWithTx", opentracing.Tags{"count": len(dss)})
	defer span.Finish()
	for _, ds := range dss {
		if err := ds.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
