package miner

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

type MinerSectorDeal struct {
	Height   int64  `pg:",pk,notnull,use_zero"`
	MinerID  string `pg:",pk,notnull"`
	SectorID uint64 `pg:",pk,use_zero"`
	DealID   uint64 `pg:",use_zero"`
}

func (ds *MinerSectorDeal) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, ds).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting marker deal sector: %v", err)
	}
	return nil
}

type MinerSectorDealList []*MinerSectorDeal

func (ml MinerSectorDealList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerSectorDealList.PersistWithTx", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()
	if len(ml) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &ml).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting miner deal sector list: %w")
	}
	return nil
}
