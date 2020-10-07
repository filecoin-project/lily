package messages

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
)

type Message struct {
	Cid string `pg:",pk,notnull"`

	From       string `pg:",notnull"`
	To         string `pg:",notnull"`
	Value      string `pg:",notnull"`
	GasFeeCap  string `pg:",notnull"`
	GasPremium string `pg:",notnull"`

	GasLimit  int64  `pg:",use_zero"`
	SizeBytes int    `pg:",use_zero"`
	Nonce     uint64 `pg:",use_zero"`
	Method    uint64 `pg:",use_zero"`

	Params []byte
}

func (m *Message) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, m).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting message: %w", err)
	}
	return nil
}

type Messages []*Message

func (ms Messages) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "Messages.PersistWithTx", trace.WithAttributes(label.Int("count", len(ms))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.TaskType, "message/message"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if _, err := tx.ModelContext(ctx, &ms).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting messages: %w", err)
	}
	return nil
}
