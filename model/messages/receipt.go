package messages

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/opentracing/opentracing-go"
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "Receipts.PersistWithTx", opentracing.Tags{"count": len(rs)})
	defer span.Finish()
	for _, r := range rs {
		if err := r.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
