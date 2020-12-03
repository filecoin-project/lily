package derived

import (
	"context"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

type GasOutputs struct {
	tableName          struct{} `pg:"derived_gas_outputs"` //nolint: structcheck,unused
	Height             int64    `pg:",pk,use_zero,notnull"`
	Cid                string   `pg:",pk,notnull"`
	StateRoot          string   `pg:",pk,notnull"`
	From               string   `pg:",notnull"`
	To                 string   `pg:",notnull"`
	Value              string   `pg:"type:numeric,notnull"`
	GasFeeCap          string   `pg:"type:numeric,notnull"`
	GasPremium         string   `pg:"type:numeric,notnull"`
	GasLimit           int64    `pg:",use_zero,notnull"`
	SizeBytes          int      `pg:",use_zero,notnull"`
	Nonce              uint64   `pg:",use_zero,notnull"`
	Method             uint64   `pg:",use_zero,notnull"`
	ActorName          string   `pg:",notnull"`
	ExitCode           int64    `pg:",use_zero,notnull"`
	GasUsed            int64    `pg:",use_zero,notnull"`
	ParentBaseFee      string   `pg:"type:numeric,notnull"`
	BaseFeeBurn        string   `pg:"type:numeric,notnull"`
	OverEstimationBurn string   `pg:"type:numeric,notnull"`
	MinerPenalty       string   `pg:"type:numeric,notnull"`
	MinerTip           string   `pg:"type:numeric,notnull"`
	Refund             string   `pg:"type:numeric,notnull"`
	GasRefund          int64    `pg:",use_zero,notnull"`
	GasBurned          int64    `pg:",use_zero,notnull"`
}

func (g *GasOutputs) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, g).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting derived gas outputs: %w", err)
	}
	return nil
}

type GasOutputsList []*GasOutputs

func (l GasOutputsList) Persist(ctx context.Context, db *pg.DB) error {
	return db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		return l.PersistWithTx(ctx, tx)
	})
}

func (l GasOutputsList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if len(l) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "GasOutputsList.PersistWithTx", trace.WithAttributes(label.Int("count", len(l))))
	defer span.End()
	if _, err := tx.ModelContext(ctx, &l).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting derived gas outputs: %w", err)
	}
	return nil
}
