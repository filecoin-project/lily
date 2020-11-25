package power

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/metrics"
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

func (cp *ChainPower) Persist(ctx context.Context, db *pg.DB) error {
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return cp.PersistWithTx(ctx, tx)
	})
}

func (cp *ChainPower) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainPower.PersistWithTx")
	defer span.End()
	if _, err := tx.ModelContext(ctx, cp).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting chain power: %w", err)
	}
	return nil
}

// ChainPowerList is a slice of ChainPowers for batch insertion.
type ChainPowerList []*ChainPower

// PersistWithTx makes a batch insertion of the list using the given
// transaction.
func (cpl ChainPowerList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ChainPowerList.PersistWithTx", trace.WithAttributes(label.Int("count", len(cpl))))
	defer span.End()

	if len(cpl) == 0 {
		return nil
	}
	if _, err := tx.ModelContext(ctx, &cpl).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting chain power list: %w", err)
	}
	return nil
}
