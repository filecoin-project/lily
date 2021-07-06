package extract

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
)

type PowerTaskFunc = func(ctx context.Context, ectx *PowerStateExtractionContext) (model.Persistable, error)

var (
	ModelTaskRegistry = map[model.Persistable]PowerTaskFunc{}
)

func Register(m model.Persistable, taskFunc PowerTaskFunc) {
	if _, ok := ModelTaskRegistry[m]; ok {
		panic("overridng previously registered task")
	}
	ModelTaskRegistry[m] = taskFunc
}

func GetModelExtractor(m model.Persistable) (PowerTaskFunc, bool) {
	e, ok := ModelTaskRegistry[m]
	return e, ok
}
