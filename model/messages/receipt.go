package messages

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

type Receipt struct {
	Message   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	Idx      int   `pg:",use_zero"`
	ExitCode int64 `pg:",use_zero"`
	GasUsed  int64 `pg:",use_zero"`

	Return []byte
}

func (r *Receipt) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, r).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting receipt: %w", err)
	}
	return nil
}

type Receipts []*Receipt

func (rs Receipts) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "Receipts.PersistWithTx", trace.WithAttributes(label.Int("count", len(rs))))
	defer span.End()
	for _, r := range rs {
		if err := r.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
