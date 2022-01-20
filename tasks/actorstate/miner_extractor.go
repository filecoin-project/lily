package actorstate

import (
	"context"
	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/ipfs/go-cid"
)

var minerAllowed map[cid.Cid]bool

func init() {
	minerAllowed = make(map[cid.Cid]bool)
	for _, c := range miner.AllCodes() {
		minerAllowed[c] = true
	}
	model.RegisterActorModelExtractor(&minermodel.MinerInfo{}, MinerInfoExtractor{})
	model.RegisterActorModelExtractor(&minermodel.MinerFeeDebt{}, MinerFeeDebtExtractor{})
	model.RegisterActorModelExtractor(&minermodel.MinerLockedFund{}, MinerLockedFundsExtractor{})
	model.RegisterActorModelExtractor(&minermodel.MinerCurrentDeadlineInfo{}, MinerCurrentDeadlineInfoExtractor{})
	model.RegisterActorModelExtractor(&minermodel.MinerPreCommitInfo{}, MinerPreCommitInfosExtractor{})
	model.RegisterActorModelExtractor(&minermodel.MinerSectorInfo{}, MinerSectorInfosExtractor{})
	model.RegisterActorModelExtractor(&minermodel.MinerSectorEvent{}, MinerSectorEventsExtractor{})
	model.RegisterActorModelExtractor(&minermodel.MinerSectorDeal{}, MinerSectorDealsExtractor{})
	model.RegisterActorModelExtractor(&minermodel.MinerSectorPost{}, MinerSectorPoStsExtractor{})
}

var _ model.ActorStateExtractor = (*MinerInfoExtractor)(nil)

type MinerInfoExtractor struct{}

func (MinerInfoExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractMinerInfo(ctx, ActorInfo(actor), ec)
}

func (MinerInfoExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerInfoExtractor) Name() string {
	return "miner_infos"
}

var _ model.ActorStateExtractor = (*MinerLockedFundsExtractor)(nil)

type MinerLockedFundsExtractor struct{}

func (MinerLockedFundsExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractMinerLockedFunds(ctx, ActorInfo(actor), ec)
}

func (MinerLockedFundsExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerLockedFundsExtractor) Name() string {
	return "miner_locked_funds"
}

var _ model.ActorStateExtractor = (*MinerFeeDebtExtractor)(nil)

type MinerFeeDebtExtractor struct{}

func (MinerFeeDebtExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractMinerFeeDebt(ctx, ActorInfo(actor), ec)
}

func (MinerFeeDebtExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerFeeDebtExtractor) Name() string {
	return "miner_fee_debts"
}

var _ model.ActorStateExtractor = (*MinerCurrentDeadlineInfoExtractor)(nil)

type MinerCurrentDeadlineInfoExtractor struct{}

func (MinerCurrentDeadlineInfoExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractMinerCurrentDeadlineInfo(ctx, ActorInfo(actor), ec)
}

func (MinerCurrentDeadlineInfoExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerCurrentDeadlineInfoExtractor) Name() string {
	return "miner_current_deadline_infos"
}

var _ model.ActorStateExtractor = (*MinerPreCommitInfosExtractor)(nil)

// TODO everything beloew this needs special performance considerations. We are calling the same method in all Extact() calls, but returning different fields. This is to stay consistenint with modle name extraction pattern
type MinerPreCommitInfosExtractor struct{}

func (MinerPreCommitInfosExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	preCommit, _, _, _, err := ExtractMinerSectorData(ctx, ec, ActorInfo(actor), api)
	return preCommit, err
}

func (MinerPreCommitInfosExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerPreCommitInfosExtractor) Name() string {
	return "miner_pre_commit_infos"
}

var _ model.ActorStateExtractor = (*MinerSectorDealsExtractor)(nil)

type MinerSectorDealsExtractor struct{}

func (MinerSectorDealsExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	_, _, sectorDeals, _, err := ExtractMinerSectorData(ctx, ec, ActorInfo(actor), api)
	return sectorDeals, err
}

func (MinerSectorDealsExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerSectorDealsExtractor) Name() string {
	return "miner_sector_deals"
}

var _ model.ActorStateExtractor = (*MinerSectorEventsExtractor)(nil)

type MinerSectorEventsExtractor struct{}

func (MinerSectorEventsExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	_, _, _, sectorEvents, err := ExtractMinerSectorData(ctx, ec, ActorInfo(actor), api)
	return sectorEvents, err
}

func (MinerSectorEventsExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerSectorEventsExtractor) Name() string {
	return "miner_sector_events"
}

var _ model.ActorStateExtractor = (*MinerSectorInfosExtractor)(nil)

type MinerSectorInfosExtractor struct{}

func (MinerSectorInfosExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	_, sectorInfos, _, _, err := ExtractMinerSectorData(ctx, ec, ActorInfo(actor), api)
	return sectorInfos, err
}

func (MinerSectorInfosExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerSectorInfosExtractor) Name() string {
	return "miner_sector_infos"
}

var _ model.ActorStateExtractor = (*MinerSectorPoStsExtractor)(nil)

type MinerSectorPoStsExtractor struct{}

func (MinerSectorPoStsExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewMinerStateExtractionContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	ai := ActorInfo(actor)
	return ExtractMinerPoSts(ctx, &ai, ec, api)
}

func (MinerSectorPoStsExtractor) Allow(code cid.Cid) bool {
	return minerAllowed[code]
}

func (MinerSectorPoStsExtractor) Name() string {
	return "miner_sector_posts"
}
