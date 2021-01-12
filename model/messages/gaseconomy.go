package messages

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MessageGasEconomy struct {
	tableName struct{} `pg:"message_gas_economy"` // nolint: structcheck,unused
	Height    int64    `pg:",pk,notnull,use_zero"`
	StateRoot string   `pg:",pk,notnull"`

	BaseFee          float64 `pg:"type:numeric,use_zero"`
	BaseFeeChangeLog float64 `pg:",use_zero"`

	GasLimitTotal       int64 `pg:"type:numeric,use_zero"`
	GasLimitUniqueTotal int64 `pg:"type:numeric,use_zero"`

	GasFillRatio     float64 `pg:",use_zero"`
	GasCapacityRatio float64 `pg:",use_zero"`
	GasWasteRatio    float64 `pg:",use_zero"`
}

func (g *MessageGasEconomy) Persist(ctx context.Context, s model.StorageBatch) error {
	return s.PersistModel(ctx, g)
}
