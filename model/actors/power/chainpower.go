package power

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
)

type ChainPower struct {
	StateRoot           string `pg:",pk"`
	NewRawBytesPower    string `pg:",notnull"`
	NewQABytesPower     string `pg:",notnull"`
	NewPledgeCollateral string `pg:",notnull"`

	TotalRawBytesPower     string `pg:",notnull"`
	TotalRawBytesCommitted string `pg:",notnull"`
	TotalQABytesPower      string `pg:",notnull"`
	TotalQABytesCommitted  string `pg:",notnull"`
	TotalPledgeCollateral  string `pg:",notnull"`

	QASmoothedPositionEstimate string `pg:",notnull"`
	QASmoothedVelocityEstimate string `pg:",notnull"`

	MinerCount                 int64 `pg:",use_zero"`
	MinimumConsensusMinerCount int64 `pg:",use_zero"`
}

func (cp *ChainPower) Persist(ctx context.Context, db *pg.DB) error {
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
