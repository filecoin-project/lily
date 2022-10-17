package tasks

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
	block2 "github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain/block"
	economics2 "github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain/economics"
	message2 "github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain/message"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/chain/tipset"
	v2 "github.com/filecoin-project/lily/model/v2"
)

type TipSetTaskTransforms struct {
	Tasks        []v2.ModelMeta
	Transformers []transform.TipSetStateHandler
}

type ActorTaskTransforms struct {
	Tasks        []v2.ModelMeta
	Transformers []transform.ActorStateHandler
}

func GetTransformersForModelMeta(meta []v2.ModelMeta) ([]transform.TipSetStateHandler, []transform.ActorStateHandler, error) {
	var ts []transform.TipSetStateHandler
	var as []transform.ActorStateHandler
	// I hate these &*$(%*$ task names _SO_ much
	for _, m := range meta {
		found := false
		for _, trns := range TipSetHandlers {
			for _, t := range trns.Transformers {
				if t.ModelType().Equals(t.ModelType()) {
					found = true
					ts = append(ts, t)
				}
			}
		}
		for _, trns := range ActorHandlers {
			for _, t := range trns.Transformers {
				if t.ModelType().Equals(t.ModelType()) {
					found = true
					as = append(as, t)
				}
			}
		}
		if !found {
			return nil, nil, fmt.Errorf("no transformer for meta %s", m.String())
		}
	}
	return ts, as, nil
}

func GetTransformersForTasks(tasks ...string) (*TipSetTaskTransforms, *ActorTaskTransforms, []v2.ModelMeta, error) {
	tst := make(map[transform.TipSetStateHandler]struct{})
	ast := make(map[transform.ActorStateHandler]struct{})
	for _, t := range tasks {
		tsTransform, tsFound := TipSetHandlers[t]
		acTransform, acFound := ActorHandlers[t]
		if !tsFound && !acFound {
			return nil, nil, nil, fmt.Errorf("no transformer for task %s", t)
		}
		if tsFound {
			for _, t := range tsTransform.Transformers {
				tst[t] = struct{}{}
			}
		}
		if acFound {
			for _, a := range acTransform.Transformers {
				ast[a] = struct{}{}
			}
		}
	}
	tasksOut := make([]v2.ModelMeta, 0, len(tasks))
	tsOut := &TipSetTaskTransforms{
		Tasks:        make([]v2.ModelMeta, 0, len(tst)),
		Transformers: make([]transform.TipSetStateHandler, 0, len(tst)),
	}
	asOut := &ActorTaskTransforms{
		Tasks:        make([]v2.ModelMeta, 0, len(ast)),
		Transformers: make([]transform.ActorStateHandler, 0, len(ast)),
	}
	for t := range tst {
		tasksOut = append(tasksOut, t.ModelType())
		tsOut.Tasks = append(tsOut.Tasks, t.ModelType())
		tsOut.Transformers = append(tsOut.Transformers, t)
	}
	for a := range ast {
		tasksOut = append(tasksOut, a.ModelType())
		asOut.Tasks = append(asOut.Tasks, a.ModelType())
		asOut.Transformers = append(asOut.Transformers, a)
	}
	return tsOut, asOut, tasksOut, nil
}

type TipSetTransforms struct {
	Transformers []transform.TipSetStateHandler
}

var TipSetHandlers = map[string]TipSetTransforms{
	BlockHeader: {
		Transformers: []transform.TipSetStateHandler{
			block2.NewBlockHeaderTransform(BlockHeader),
		},
	},
	BlockParent: {
		Transformers: []transform.TipSetStateHandler{
			block2.NewBlockParentsTransform(BlockParent),
		},
	},
	DrandBlockEntrie: {
		Transformers: []transform.TipSetStateHandler{
			block2.NewDrandBlockEntryTransform(DrandBlockEntrie),
		},
	},
	Message: {
		Transformers: []transform.TipSetStateHandler{
			message2.NewMessageTransform(Message),
		},
	},
	BlockMessage: {
		Transformers: []transform.TipSetStateHandler{
			message2.NewBlockMessageTransform(BlockMessage),
		},
	},
	Receipt: {
		Transformers: []transform.TipSetStateHandler{
			message2.NewReceiptTransform(Receipt),
		},
	},
	MessageGasEconomy: {
		Transformers: []transform.TipSetStateHandler{
			economics2.NewGasEconomyTransform(MessageGasEconomy),
		},
	},
	ParsedMessage: {
		Transformers: []transform.TipSetStateHandler{
			message2.NewParsedMessageTransform(ParsedMessage),
		},
	},
	GasOutputs: {
		Transformers: []transform.TipSetStateHandler{
			message2.NewGasOutputTransform(GasOutputs),
		},
	},
	VmMessage: {
		Transformers: []transform.TipSetStateHandler{
			message2.NewVMMessageTransform(VmMessage),
		},
	},
	InternalMessage: {
		Transformers: []transform.TipSetStateHandler{
			message2.NewImplicitMessageTransform(InternalMessage),
		},
	},
	InternalParsedMessage: {
		Transformers: []transform.TipSetStateHandler{
			message2.NewImplicitParsedMessageTransform(InternalParsedMessage),
		},
	},
	ChainEconomics: {
		Transformers: []transform.TipSetStateHandler{
			economics2.NewCirculatingSupplyTransform(ChainEconomics),
		},
	},
	ChainConsensus: {
		Transformers: []transform.TipSetStateHandler{
			tipset.NewConsensusTransform(ChainConsensus),
		},
	},
}

type ActorStateTransforms struct {
	Transformers []transform.ActorStateHandler
}

var ActorHandlers = map[string]ActorStateTransforms{
	MinerSectorDeal: {
		Transformers: []transform.ActorStateHandler{
			miner.NewSectorDealsTransformer(MinerSectorDeal),
		},
	},
	MinerSectorEvent: {
		Transformers: []transform.ActorStateHandler{
			miner.NewSectorEventTransformer(MinerSectorEvent),
			miner.NewPrecommitEventTransformer(MinerSectorEvent),
		},
	},
	MinerPreCommitInfo: {
		Transformers: []transform.ActorStateHandler{
			miner.NewPrecommitInfoTransformer(MinerPreCommitInfo),
		},
	},
	MinerSectorInfoV7: {
		Transformers: []transform.ActorStateHandler{
			miner.NewSectorInfoTransform(MinerSectorInfoV7),
		},
	},
	MinerInfo: {
		Transformers: []transform.ActorStateHandler{
			miner.NewMinerInfoTransform(MinerInfo),
		},
	},
	MinerLockedFund: {
		Transformers: []transform.ActorStateHandler{
			miner.NewFundsTransform(MinerLockedFund),
		},
	},
	MinerCurrentDeadlineInfo: {
		Transformers: []transform.ActorStateHandler{
			miner.NewDeadlineInfoTransform(MinerCurrentDeadlineInfo),
		},
	},
	MinerFeeDebt: {
		Transformers: []transform.ActorStateHandler{
			miner.NewFeeDebtTransform(MinerFeeDebt),
		},
	},
	MinerSectorPost: {
		Transformers: []transform.ActorStateHandler{
			miner.NewPostSectorMessageTransform(MinerSectorPost),
		},
	},
	MarketDealState: {
		Transformers: []transform.ActorStateHandler{
			market.NewDealStateTransformer(MarketDealState),
		},
	},
	MarketDealProposal: {
		Transformers: []transform.ActorStateHandler{
			market.NewDealProposalTransformer(MarketDealProposal),
		},
	},
	Actor: {
		Transformers: []transform.ActorStateHandler{
			raw.NewActorTransform(Actor),
		},
	},
	ActorState: {
		Transformers: []transform.ActorStateHandler{
			raw.NewActorStateTransform(ActorState),
		},
	},
	IdAddress: {
		Transformers: []transform.ActorStateHandler{
			init_.NewIDAddressTransform(IdAddress),
		},
	},
	MultisigTransaction: {
		Transformers: []transform.ActorStateHandler{
			multisig.NewTransactionTransform(MultisigTransaction),
		},
	},
	ChainPower: {
		Transformers: []transform.ActorStateHandler{
			power.NewChainPowerTransform(ChainPower),
		},
	},
	PowerActorClaim: {
		Transformers: []transform.ActorStateHandler{
			power.NewClaimedPowerTransform(PowerActorClaim),
		},
	},
	ChainReward: {
		Transformers: []transform.ActorStateHandler{
			reward.NewChainRewardTransform(ChainReward),
		},
	},
	VerifiedRegistryVerifiedClient: {
		Transformers: []transform.ActorStateHandler{
			verifreg.NewVerifiedClientTransform(VerifiedRegistryVerifiedClient),
		},
	},
	VerifiedRegistryVerifier: {
		Transformers: []transform.ActorStateHandler{
			verifreg.NewVerifierTransform(VerifiedRegistryVerifier),
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
