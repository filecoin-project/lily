package market

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model/registry"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/actor"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/market/extract"
	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	market "github.com/filecoin-project/sentinel-visor/chain/actors/builtin/market"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

const ActorStatesMarketTask = "actorstatesmarket" // task that only extracts market actor states (but not the raw state)

func init() {
	for _, c := range market.AllCodes() {
		actor.Register(c, StorageMarketExtractor{})
	}
	for m := range extract.ModelTaskRegistry {
		registry.ModelRegistry.Register(ActorStatesMarketTask, m)
	}
}

// StorageMarketExtractor extracts market actor state
type StorageMarketExtractor struct{}

func (m StorageMarketExtractor) Extract(ctx context.Context, a actor.ActorInfo, node actor.ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMarketExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	ec, err := extract.NewMarketStateExtractionContext(ctx, a, node)
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
