package actorstate

import (
	"context"
	init_ "github.com/filecoin-project/lily/chain/actors/builtin/init"
	"github.com/filecoin-project/lily/model"
	initmodel "github.com/filecoin-project/lily/model/actors/init"
	"github.com/ipfs/go-cid"
)

var initAllowed map[cid.Cid]bool

func init() {
	initAllowed = make(map[cid.Cid]bool)
	for _, c := range init_.AllCodes() {
		initAllowed[c] = true
	}
	model.RegisterActorModelExtractor(&initmodel.IdAddress{}, IdAddressExtractor{})
}

var _ model.ActorStateExtractor = (*IdAddressExtractor)(nil)

type IdAddressExtractor struct{}

func (IdAddressExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	return InitExtractor{}.Extract(ctx, ActorInfo(actor), api)
}

func (IdAddressExtractor) Allow(code cid.Cid) bool {
	return initAllowed[code]
}

func (IdAddressExtractor) Name() string {
	return "id_addresses"
}
