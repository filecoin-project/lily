package power

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model/registry"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/actor"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/power/extract"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/power"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

// was services/processor/tasks/power/power.go

// StoragePowerExtractor extracts power actor state
type StoragePowerExtractor struct{}

const ActorStatesPowerTask = "actorstatespower" // task that only extracts power actor states (but not the raw state)

func init() {
	for _, c := range power.AllCodes() {
		actor.Register(c, StoragePowerExtractor{})
	}
	for m := range extract.ModelTaskRegistry {
		registry.ModelRegistry.Register(ActorStatesPowerTask, m)
	}
}

func (StoragePowerExtractor) Extract(ctx context.Context, a actor.ActorInfo, node actor.ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "StoragePowerExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	ec, err := extract.NewPowerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, err
	}

	var out model.PersistableList
	for _, m := range a.Models {
		extf, found := extract.GetModelExtractor(m)
		if !found {
			return nil, xerrors.Errorf("failed to find extractor for: %T", m)
		}
		data, err := extf(ctx, ec)
		if err != nil {
			return nil, err
		}
		out = append(out, data)
	}
	return out, nil
}
