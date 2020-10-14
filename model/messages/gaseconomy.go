package messages

import (
	"context"

	"github.com/go-pg/pg/v10"
	"golang.org/x/xerrors"
)

type MessageGasEconomy struct {
	tableName struct{} `pg:"message_gas_economy"`
	StateRoot string   `pg:",pk,notnull"`

	BaseFee          float64 `pg:",use_zero"`
	BaseFeeChangeLog float64 `pg:",use_zero"`

	GasLimitTotal       int64 `pg:",use_zero"`
	GasLimitUniqueTotal int64 `pg:",use_zero"`

	GasFillRatio     float64 `pg:",use_zero"`
	GasCapacityRatio float64 `pg:",use_zero"`
	GasWasteRatio    float64 `pg:",use_zero"`
}

func (g *MessageGasEconomy) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, g).
		OnConflict("do nothing").
		Insert(); err != nil {
		return xerrors.Errorf("persisting derived gas economy: %w", err)
	}
	return nil
}
