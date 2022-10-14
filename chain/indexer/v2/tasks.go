package v2

import (
	"fmt"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/init_"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/market"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/miner"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/multisig"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/power"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/raw"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/reward"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor/verifreg"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/block"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/economics"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/message"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/tipset"
	v2 "github.com/filecoin-project/lily/model/v2"
)

type TaskMeta struct {
	Transformers []transform.Handler
}

type ThingIDK struct {
	Tasks        []v2.ModelMeta
	Transformers []transform.Handler
}

func GetTransformersForTasks(tasks ...string) (*ThingIDK, error) {
	tmp := make(map[transform.Handler]struct{})
	for _, t := range tasks {
		meta, ok := TaskHandlers[t]
		if !ok {
			return nil, fmt.Errorf("no transformer for task %s", t)
		}
		for _, transformer := range meta.Transformers {
			tmp[transformer] = struct{}{}
		}
	}
	out := &ThingIDK{
		Tasks:        make([]v2.ModelMeta, 0, 10),
		Transformers: make([]transform.Handler, 0, 10),
	}
	for t := range tmp {
		out.Tasks = append(out.Tasks, t.ModelType())
		out.Transformers = append(out.Transformers, t)
	}
	return out, nil
}

func GetLegacyTaskNameForTransform(name string) string {
	for task, transformes := range TaskHandlers {
		for _, handler := range transformes.Transformers {
			if name == handler.Name() {
				return task
			}
		}
	}
	panic("developer error")
}

var TaskHandlers = map[string]TaskMeta{
	BlockHeader: {
		Transformers: []transform.Handler{
			block.NewBlockHeaderTransform(),
		},
	},
	BlockParent: {
		Transformers: []transform.Handler{
			block.NewBlockParentsTransform(),
		},
	},
	DrandBlockEntrie: {
		Transformers: []transform.Handler{
			block.NewDrandBlockEntryTransform(),
		},
	},
	MinerSectorDeal: {
		Transformers: []transform.Handler{
			miner.NewSectorDealsTransformer(),
		},
	},
	MinerSectorEvent: {
		Transformers: []transform.Handler{
			miner.NewSectorEventTransformer(),
			miner.NewPrecommitEventTransformer(),
		},
	},
	MinerPreCommitInfo: {
		Transformers: []transform.Handler{
			miner.NewPrecommitInfoTransformer(),
		},
	},
	MinerSectorInfoV7: {
		Transformers: []transform.Handler{
			miner.NewSectorInfoTransform(),
		},
	},
	MinerInfo: {
		Transformers: []transform.Handler{
			miner.NewMinerInfoTransform(),
		},
	},
	MinerLockedFund: {
		Transformers: []transform.Handler{
			miner.NewFundsTransform(),
		},
	},
	MinerCurrentDeadlineInfo: {
		Transformers: []transform.Handler{
			miner.NewDeadlineInfoTransform(),
		},
	},
	MinerFeeDebt: {
		Transformers: []transform.Handler{
			miner.NewFeeDebtTransform(),
		},
	},
	MinerSectorPost: {
		Transformers: []transform.Handler{
			miner.NewPostSectorMessageTransform(),
		},
	},
	MarketDealState: {
		Transformers: []transform.Handler{
			market.NewDealStateTransformer(),
		},
	},
	MarketDealProposal: {
		Transformers: []transform.Handler{
			market.NewDealProposalTransformer(),
		},
	},
	Message: {
		Transformers: []transform.Handler{
			message.NewMessageTransform(),
		},
	},
	BlockMessage: {
		Transformers: []transform.Handler{
			message.NewBlockMessageTransform(),
		},
	},
	Receipt: {
		Transformers: []transform.Handler{
			message.NewReceiptTransform(),
		},
	},
	MessageGasEconomy: {
		Transformers: []transform.Handler{
			economics.NewGasEconomyTransform(),
		},
	},
	ParsedMessage: {
		Transformers: []transform.Handler{
			message.NewParsedMessageTransform(),
		},
	},
	GasOutputs: {
		Transformers: []transform.Handler{
			message.NewGasOutputTransform(),
		},
	},
	VmMessage: {
		Transformers: []transform.Handler{
			message.NewVMMessageTransform(),
		},
	},
	Actor: {
		Transformers: []transform.Handler{
			raw.NewActorTransform(),
		},
	},
	ActorState: {
		Transformers: []transform.Handler{
			raw.NewActorStateTransform(),
		},
	},
	IdAddress: {
		Transformers: []transform.Handler{
			init_.NewIDAddressTransform(),
		},
	},
	MultisigTransaction: {
		Transformers: []transform.Handler{
			multisig.NewTransactionTransform(),
		},
	},
	ChainPower: {
		Transformers: []transform.Handler{
			power.NewChainPowerTransform(),
		},
	},
	PowerActorClaim: {
		Transformers: []transform.Handler{
			power.NewClaimedPowerTransform(),
		},
	},
	ChainReward: {
		Transformers: []transform.Handler{
			reward.NewChainRewardTransform(),
		},
	},
	VerifiedRegistryVerifiedClient: {
		Transformers: []transform.Handler{
			verifreg.NewVerifiedClientTransform(),
		},
	},
	VerifiedRegistryVerifier: {
		Transformers: []transform.Handler{
			verifreg.NewVerifierTransform(),
		},
	},
	InternalMessage: {
		Transformers: []transform.Handler{
			message.NewImplicitMessageTransform(),
		},
	},
	InternalParsedMessage: {
		Transformers: []transform.Handler{
			message.NewImplicitParsedMessageTransform(),
		},
	},
	ChainEconomics: {
		Transformers: []transform.Handler{
			economics.NewCirculatingSupplyTransform(),
		},
	},
	ChainConsensus: {
		Transformers: []transform.Handler{
			tipset.NewConsensusTransform(),
		},
	},
}

const (
	BlockHeader      = "block_header"
	BlockParent      = "block_parent"
	DrandBlockEntrie = "drand_block_entrie"

	MarketDealProposal = "market_deal_proposal"
	MarketDealState    = "market_deal_state"

	MinerSectorDeal          = "miner_sector_deal"
	MinerSectorInfoV7        = "miner_sector_infos_v7"
	MinerSectorInfoV1_6      = "miner_sector_infos"
	MinerSectorPost          = "miner_sector_post"
	MinerPreCommitInfo       = "miner_pre_commit_info"
	MinerSectorEvent         = "miner_sector_event"
	MinerCurrentDeadlineInfo = "miner_current_deadline_info"
	MinerFeeDebt             = "miner_fee_debt"
	MinerLockedFund          = "miner_locked_fund"
	MinerInfo                = "miner_info"

	Message           = "message"
	BlockMessage      = "block_message"
	Receipt           = "receipt"
	MessageGasEconomy = "message_gas_economy"
	ParsedMessage     = "parsed_message"
	GasOutputs        = "derived_gas_outputs"

	VmMessage = "vm_messages"

	Actor      = "actor"
	ActorState = "actor_state"

	InternalMessage       = "internal_messages"
	InternalParsedMessage = "internal_parsed_messages"

	MultisigTransaction = "multisig_transaction"
	MultisigApproval    = "multisig_approvals"

	ChainPower      = "chain_power"
	PowerActorClaim = "power_actor_claim"

	ChainReward = "chain_reward"

	IdAddress = "id_address"

	ChainEconomics = "chain_economics"

	ChainConsensus = "chain_consensus"

	VerifiedRegistryVerifier       = "verified_registry_verifier"
	VerifiedRegistryVerifiedClient = "verified_registry_verified_client"
)
