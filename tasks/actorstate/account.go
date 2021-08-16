package actorstate

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/chain/actors/builtin/account"
	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
)

// AccountExtractor is a state extractor that deals with Account actors.
type AccountExtractor struct{}

func init() {
	for _, c := range account.AllCodes() {
		Register(c, AccountExtractor{})
	}
}

// Extract will create persistable data for a given actor's state.
func (AccountExtractor) Extract(ctx context.Context, a ActorInfo, emsgs []*lens.ExecutedMessage, node ActorStateAPI) (model.Persistable, error) {
	return model.NoData, nil
}

var _ ActorStateExtractor = (*AccountExtractor)(nil)
