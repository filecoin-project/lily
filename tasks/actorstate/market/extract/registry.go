package extract

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MarketTaskFunc = func(ctx context.Context, ectx *MarketStateExtractionContext) (model.Persistable, error)

var (
	ModelTaskRegistry = map[model.Persistable]MarketTaskFunc{}
)

func Register(m model.Persistable, taskFunc MarketTaskFunc) {
	if _, ok := ModelTaskRegistry[m]; ok {
		panic("overridng previously registered task")
	}
	ModelTaskRegistry[m] = taskFunc
}

func GetModelExtractor(m model.Persistable) (MarketTaskFunc, bool) {
	e, ok := ModelTaskRegistry[m]
	return e, ok
}
