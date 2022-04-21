package tasktype

const (
	ActorStatesRawTask      = "actorstatesraw"      // task that only extracts raw actor state
	ActorStatesPowerTask    = "actorstatespower"    // task that only extracts power actor states (but not the raw state)
	ActorStatesRewardTask   = "actorstatesreward"   // task that only extracts reward actor states (but not the raw state)
	ActorStatesMinerTask    = "actorstatesminer"    // task that only extracts miner actor states (but not the raw state)
	ActorStatesInitTask     = "actorstatesinit"     // task that only extracts init actor states (but not the raw state)
	ActorStatesMarketTask   = "actorstatesmarket"   // task that only extracts market actor states (but not the raw state)
	ActorStatesMultisigTask = "actorstatesmultisig" // task that only extracts multisig actor states (but not the raw state)
	ActorStatesVerifreg     = "actorstatesverifreg" // task that only extracts verified registry actor states (but not the raw state)
	BlocksTask              = "blocks"              // task that extracts block data
	MessagesTask            = "messages"            // task that extracts message data
	ChainEconomicsTask      = "chaineconomics"      // task that extracts chain economics data
	MultisigApprovalsTask   = "msapprovals"         // task that extracts multisig actor approvals
	ImplicitMessageTask     = "implicitmessage"     // task that extract implicitly executed messages: cron tick and block reward.
	ChainConsensusTask      = "consensus"
)

var TaskLookup = map[string][]string{
	ActorStatesRawTask: {
		Actor,
		ActorState,
	},
	ActorStatesPowerTask: {
		ChainPower,
		PowerActorClaim,
	},
	ActorStatesRewardTask: {
		ChainReward,
	},
	ActorStatesMinerTask: {
		MinerSectorDeal,
		MinerSectorInfoV7,
		MinerSectorInfoV1_6,
		MinerSectorPost,
		MinerPreCommitInfo,
		MinerSectorEvent,
		MinerCurrentDeadlineInfo,
		MinerFeeDebt,
		MinerLockedFund,
		MinerInfo,
	},
	ActorStatesInitTask: {
		IdAddress,
	},
	ActorStatesMarketTask: {
		MarketDealProposal,
		MarketDealState,
	},
	ActorStatesMultisigTask: {
		MultisigTransaction,
	},
	ActorStatesVerifreg: {
		VerifiedRegistryVerifier,
		VerifiedRegistryVerifiedClient,
	},
	BlocksTask: {
		BlockHeader,
		BlockParent,
		DrandBlockEntrie,
	},
	MessagesTask: {
		Message,
		Receipt,
		GasOutputs,
		MessageGasEconomy,
		BlockMessage,
	},
	ChainEconomicsTask: {
		ChainEconomics,
	},
	MultisigApprovalsTask: {
		MultisigApproval,
	},
	ImplicitMessageTask: {
		InternalMessage,
		InternalParsedMessage,
	},
	ChainConsensusTask: {
		ChainConsensus,
	},
}
