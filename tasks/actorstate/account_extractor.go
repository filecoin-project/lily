package actorstate

import (
	"context"
	"github.com/filecoin-project/lily/chain/actors/builtin/account"
	"github.com/filecoin-project/lily/model"
	"github.com/ipfs/go-cid"
)

var accountAllowed map[cid.Cid]bool

func init() {
	accountAllowed = make(map[cid.Cid]bool)
	for _, c := range account.AllCodes() {
		accountAllowed[c] = true
	}
	// TODO fix me
	model.RegisterActorModelExtractor(nil, AccountActorExtractor{})
}

var _ model.ActorStateExtractor = (*AccountActorExtractor)(nil)

type AccountActorExtractor struct{}

func (AccountActorExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	return AccountExtractor{}.Extract(ctx, ActorInfo(actor), api)
}

func (AccountActorExtractor) Allow(code cid.Cid) bool {
	return accountAllowed[code]
}

func (AccountActorExtractor) Name() string {
	return "account"
}
