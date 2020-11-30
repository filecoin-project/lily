package actorstate

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/sentinel-visor/model"
	builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

// AccountExtractor is a state extractor that deals with Account actors.
type AccountExtractor struct{}

func init() {
	Register(builtin.AccountActorCodeID, AccountExtractor{})
}

var includeAddrs = []address.Address{builtin.BurntFundsActorAddr}

// Extract will create persistable data for a given actor's state.
func (AccountExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	return model.NoData, nil
}

// Filter determines which actors this extractor is willing to extract.
func (AccountExtractor) Filter(info ActorInfo) bool {
	for _, a := range includeAddrs {
		if a == info.Address {
			return true
		}
	}
	return false
}

var _ ActorStateExtractor = (*AccountExtractor)(nil)
var _ FilteredExtractor = (*AccountExtractor)(nil)
