package miner

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/opentracing/opentracing-go"
)

type MinerDealSector struct {
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,use_zero"`
	DealID   uint64 `pg:",use_zero"`
}

func (ds *MinerDealSector) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, ds).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting marker deal sector: %v", err)
	}
	return nil
}

type MinerDealSectors []*MinerDealSector

func (dss MinerDealSectors) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "MinerDealSectors.PersistWithTx", opentracing.Tags{"count": len(dss)})
	defer span.Finish()
	for _, ds := range dss {
		if err := ds.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
