package chain

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

type ChainEconomics struct {
	tableName       struct{} `pg:"chain_economics"`
	ParentStateRoot string   `pg:",notnull"`
	CirculatingFil  string   `pg:",notnull"`
	VestedFil       string   `pg:",notnull"`
	MinedFil        string   `pg:",notnull"`
	BurntFil        string   `pg:",notnull"`
	LockedFil       string   `pg:",notnull"`
}

func (c *ChainEconomics) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, c).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting chain economics: %w", err)
	}
	return nil
}

type ChainEconomicsList []*ChainEconomics

func (l ChainEconomicsList) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return l.PersistWithTx(ctx, tx)
	})
}

func (l ChainEconomicsList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ChainEconomicsList.PersistWithTx", trace.WithAttributes(label.Int("count", len(l))))
	defer span.End()
	if _, err := tx.ModelContext(ctx, &l).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting derived gas outputs: %w", err)
	}
	return nil
}
