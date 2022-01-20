package chain

import (
	"context"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/actors/common"
	initmodel "github.com/filecoin-project/lily/model/actors/init"
	marketmodel "github.com/filecoin-project/lily/model/actors/market"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	multisigmodel "github.com/filecoin-project/lily/model/actors/multisig"
	powermodel "github.com/filecoin-project/lily/model/actors/power"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	"github.com/filecoin-project/lily/model/blocks"
	chainmodel "github.com/filecoin-project/lily/model/chain"
	"github.com/filecoin-project/lily/model/derived"
	messagemodel "github.com/filecoin-project/lily/model/messages"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

// TODO use go-pg to read the table names or code gen this
func StringToModelTypeAndExtractor(name string) (model.Persistable, ExtractorType, error) {
	switch name {
	case "block_headers":
		return &blocks.BlockHeader{}, TipSetStateExtractorType, nil
	case "block_parents":
		return &blocks.BlockParent{}, TipSetStateExtractorType, nil
	case "drand_block_entries":
		return &blocks.DrandBlockEntrie{}, TipSetStateExtractorType, nil
	case "chain_economics":
		return &chainmodel.ChainEconomics{}, TipSetStateExtractorType, nil
	case "internal_messages":
		return &messagemodel.InternalMessage{}, TipSetStateExtractorType, nil
	case "internal_parsed_messages":
		return &messagemodel.InternalParsedMessage{}, TipSetStateExtractorType, nil
	case "messages":
		return &messagemodel.Message{}, TipSetStateExtractorType, nil
	case "receipts":
		return &messagemodel.Receipt{}, TipSetStateExtractorType, nil
	case "parsed_messages":
		return &messagemodel.ParsedMessage{}, TipSetStateExtractorType, nil
	case "message_gas_economy":
		return &messagemodel.MessageGasEconomy{}, TipSetStateExtractorType, nil
	case "derived_gas_outputs":
		return &derived.GasOutputs{}, TipSetStateExtractorType, nil
	case "block_messages":
		return &messagemodel.BlockMessage{}, TipSetStateExtractorType, nil
	case "chain_consensus":
		return &chainmodel.ChainConsensus{}, TipSetStateExtractorType, nil

	case "actors":
		return &common.Actor{}, ActorStateExtractorType, nil
	case "actor_states":
		return &common.ActorState{}, ActorStateExtractorType, nil
	case "chain_powers":
		return &powermodel.ChainPower{}, ActorStateExtractorType, nil
	case "power_actor_claims":
		return &powermodel.PowerActorClaim{}, ActorStateExtractorType, nil
	case "verified_registry_verifiers":
		return &verifregmodel.VerifiedRegistryVerifier{}, ActorStateExtractorType, nil
	case "verified_registry_verified_clients":
		return &verifregmodel.VerifiedRegistryVerifiedClient{}, ActorStateExtractorType, nil
	case "chain_rewards":
		return &rewardmodel.ChainReward{}, ActorStateExtractorType, nil
	case "multisig_transactions":
		return &multisigmodel.MultisigTransaction{}, ActorStateExtractorType, nil
	case "miner_fee_debt":
		return &minermodel.MinerFeeDebt{}, ActorStateExtractorType, nil
	case "miner_infos":
		return &minermodel.MinerInfo{}, ActorStateExtractorType, nil
	case "miner_locked_funds":
		return &minermodel.MinerLockedFund{}, ActorStateExtractorType, nil
	case "miner_pre_commit_infos":
		return &minermodel.MinerPreCommitInfo{}, ActorStateExtractorType, nil
	case "miner_sector_deals":
		return &minermodel.MinerSectorDeal{}, ActorStateExtractorType, nil
	case "miner_sector_events":
		return &minermodel.MinerSectorEvent{}, ActorStateExtractorType, nil
	case "miner_sector_infos":
		return &minermodel.MinerSectorInfo{}, ActorStateExtractorType, nil
	case "miner_sector_posts":
		return &minermodel.MinerSectorPost{}, ActorStateExtractorType, nil
	case "miner_current_deadline_infos":
		return &minermodel.MinerCurrentDeadlineInfo{}, ActorStateExtractorType, nil
	case "market_deal_proposals":
		return &marketmodel.MarketDealProposal{}, ActorStateExtractorType, nil
	case "market_deal_states":
		return &marketmodel.MarketDealState{}, ActorStateExtractorType, nil
	case "id_addresses":
		return &initmodel.IdAddress{}, ActorStateExtractorType, nil
	default:
		return nil, UnknownStateExtractorType, xerrors.Errorf("unknown model name %s", name)
	}
}

type ExtractorType string

var UnknownStateExtractorType ExtractorType = "Unknown"
var TipSetStateExtractorType ExtractorType = "TipSetStateExtractor"
var ActorStateExtractorType ExtractorType = "ActorStateExtractor"

// TODO remove these and use the model interfaces if that doesn't make circular deps
type TipSetStateExtractor interface {
	Extract(ctx context.Context, current, previous *types.TipSet, api model.TipSetStateAPI) (model.Persistable, error)
	Name() string
}

type ActorStateExtractor interface {
	Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error)
	Allow(code cid.Cid) bool
	Name() string
}
