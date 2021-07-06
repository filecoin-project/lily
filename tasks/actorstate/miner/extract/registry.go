package extract

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type MinerTaskFunc = func(ctx context.Context, ectx *MinerStateExtractionContext) (model.Persistable, error)

var (
	ModelTaskRegistry = map[model.Persistable]MinerTaskFunc{}
)

func Register(m model.Persistable, taskFunc MinerTaskFunc) {
	if _, ok := ModelTaskRegistry[m]; ok {
		panic("overridng previously registered task")
	}
	ModelTaskRegistry[m] = taskFunc
}

func GetModelExtractor(m model.Persistable) (MinerTaskFunc, bool) {
	e, ok := ModelTaskRegistry[m]
	return e, ok
}
