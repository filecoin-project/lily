package account

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/account"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/actor"
)

// AccountExtractor is a state extractor that deals with Account actors.
type AccountExtractor struct{}

func init() {
	for _, c := range account.AllCodes() {
		actor.Register(c, AccountExtractor{})
	}
}

// Extract will create persistable data for a given actor's state.
func (AccountExtractor) Extract(ctx context.Context, a actor.ActorInfo, node actor.ActorStateAPI) (model.Persistable, error) {
	return model.NoData, nil
}

var _ actor.ActorStateExtractor = (*AccountExtractor)(nil)
