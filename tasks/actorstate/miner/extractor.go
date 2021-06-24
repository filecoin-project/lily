package miner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/miner"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/model/registry"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner/tasks"

	// This makes me sad, but seems like a reasonable trade off, ensures the extractor have registered themselves before the below init method is called.
	_ "github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner/tasks/extractors"
)

const ActorStatesMinerTask = "actorstatesminer" // task that only extracts miner actor states (but not the raw state)

func init() {
	// register this extractor as being responsible for the miner actor codes
	for _, c := range miner.AllCodes() {
		actorstate.Register(c, StorageMinerExtractor{})
	}
	// register this task being responsible for producing the following models.
	for m := range tasks.ModelTaskRegistry {
		registry.ModelRegistry.Register(ActorStatesMinerTask, m)
	}
	fmt.Println(tasks.ModelTaskRegistry)
}

// StorageMinerExtractor extracts miner actor state
type StorageMinerExtractor struct{}

func (m StorageMinerExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "StorageMinerExtractor")
	if span.IsRecording() {
		span.SetAttributes(label.String("actor", a.Address.String()))
	}
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	ec, err := tasks.NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("creating miner state extraction context: %w", err)
	}

	var out model.PersistableList
	for _, m := range a.Models {
		extF, found := tasks.GetModelExtractor(m)
		if !found {
			return nil, xerrors.Errorf("Failed to find extractor for: %T", m)
		}
		data, err := extF(ctx, ec)
		if err != nil {
			return nil, err
		}
		out = append(out, data)
	}
	return out, nil
}
