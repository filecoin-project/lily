package actorstate

import (
	"context"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	"github.com/filecoin-project/lily/model"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	"github.com/ipfs/go-cid"
)

var marketAllowed map[cid.Cid]bool

func init() {
	marketAllowed = make(map[cid.Cid]bool)
	for _, c := range market.AllCodes() {
		marketAllowed[c] = true
	}
	model.RegisterActorModelExtractor(&marketmodel.MarketDealProposal{}, MarketDealProposalsExtractor{})
	model.RegisterActorModelExtractor(&marketmodel.MarketDealState{}, MarketDealStatesExtractor{})
}

var _ model.ActorStateExtractor = (*MarketDealProposalsExtractor)(nil)

type MarketDealProposalsExtractor struct{}

func (MarketDealProposalsExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMarketStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractMarketDealStates(ctx, ec)
}

func (MarketDealProposalsExtractor) Allow(code cid.Cid) bool {
	return marketAllowed[code]
}

func (MarketDealProposalsExtractor) Name() string {
	return "market_deal_proposals"
}

var _ model.ActorStateExtractor = (*MarketDealStatesExtractor)(nil)

type MarketDealStatesExtractor struct{}

func (MarketDealStatesExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMarketStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractMarketDealStates(ctx, ec)
}

func (MarketDealStatesExtractor) Allow(code cid.Cid) bool {
	return marketAllowed[code]
}

func (MarketDealStatesExtractor) Name() string {
	return "market_deal_states"
}
