package actorstate

import (
	"context"
	"github.com/filecoin-project/lily/chain/actors/builtin/multisig"
	"github.com/filecoin-project/lily/model"
	multisigmodel "github.com/filecoin-project/lily/model/actors/multisig"
	"github.com/ipfs/go-cid"
)

var multisigAllowed map[cid.Cid]bool

func init() {
	multisigAllowed = make(map[cid.Cid]bool)
	for _, c := range multisig.AllCodes() {
		multisigAllowed[c] = true
	}
	model.RegisterActorModelExtractor(&multisigmodel.MultisigTransaction{}, MultisigTransactionsExtractor{})
}

var _ model.ActorStateExtractor = (*MultisigTransactionsExtractor)(nil)

type MultisigTransactionsExtractor struct{}

func (MultisigTransactionsExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMultiSigExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractMultisigTransactions(ctx, ActorInfo(actor), ec)
}

func (MultisigTransactionsExtractor) Allow(code cid.Cid) bool {
	return multisigAllowed[code]
}

func (MultisigTransactionsExtractor) Name() string {
	return "power_actor_claims"
}
