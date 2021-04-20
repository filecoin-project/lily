package actorstate

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/model"
	sa0builtin "github.com/filecoin-project/specs-actors/actors/builtin"
	sa2builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	sa3builtin "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	sa4builtin "github.com/filecoin-project/specs-actors/v4/actors/builtin"
)

// AccountExtractor is a state extractor that deals with Account actors.
type AccountExtractor struct{}

func init() {
	Register(sa0builtin.AccountActorCodeID, AccountExtractor{})
	Register(sa2builtin.AccountActorCodeID, AccountExtractor{})
	Register(sa3builtin.AccountActorCodeID, AccountExtractor{})
	Register(sa4builtin.AccountActorCodeID, AccountExtractor{})
}

// Extract will create persistable data for a given actor's state.
func (AccountExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	return model.NoData, nil
}

var _ ActorStateExtractor = (*AccountExtractor)(nil)
