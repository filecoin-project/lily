package actorstate

import (
	"context"
	"github.com/filecoin-project/lily/chain/actors/builtin/power"
	"github.com/filecoin-project/lily/model"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	"github.com/ipfs/go-cid"
)

var powerAllowed map[cid.Cid]bool

func init() {
	powerAllowed = make(map[cid.Cid]bool)
	for _, c := range power.AllCodes() {
		powerAllowed[c] = true
	}
	model.RegisterActorModelExtractor(&powermodel.ChainPower{}, PowerActorClaimsExtractor{})
	model.RegisterActorModelExtractor(&powermodel.PowerActorClaim{}, ChainPowerExtractor{})
}

var _ model.ActorStateExtractor = (*PowerActorClaimsExtractor)(nil)

type PowerActorClaimsExtractor struct{}

func (PowerActorClaimsExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewPowerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractClaimedPower(ctx, ec)
}

func (PowerActorClaimsExtractor) Allow(code cid.Cid) bool {
	return powerAllowed[code]
}

func (PowerActorClaimsExtractor) Name() string {
	return "power_actor_claims"
}

var _ model.ActorStateExtractor = (*ChainPowerExtractor)(nil)

type ChainPowerExtractor struct{}

func (ChainPowerExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewPowerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractChainPower(ec)
}

func (ChainPowerExtractor) Allow(code cid.Cid) bool {
	return powerAllowed[code]
}

func (ChainPowerExtractor) Name() string {
	return "chain_powers"
}
