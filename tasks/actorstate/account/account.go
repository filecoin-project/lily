package account

import (
	"context"

	"go.opentelemetry.io/otel"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

var _ actorstate.ActorStateExtractor = (*AccountExtractor)(nil)

// AccountExtractor is a state extractor that deals with Account actors.
type AccountExtractor struct{}

// Extract will create persistable data for a given actor's state.
func (AccountExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	_, span := otel.Tracer("").Start(ctx, "AccountExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}
	return model.NoData, nil
}
