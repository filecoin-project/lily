package indexer

const (
	// blocks
	BlocksTask = "blocks"

	// consensus
	ChainConsensusTask = "chain_consensus"

	// economy
	ChainEconomicsTask    = "chain_economics"
	DerivedGasOutputsTask = "derived_gas_outputs"

	// messages
	MessageGasEconomyTask = "message_gas_economy"
	MessagesTask          = "messages"
	BlockMessagesTask     = "block_messages"
	ParsedMessageTask     = "parsed_messages"

	// internal messages
	InternalMessagesTask       = "internal_messages"
	InternalParsedMessagesTask = "internal_parsed_messages"

	// receipts
	ReceiptTask = "receipts"

	// multisig approval messages
	MultiSigApprovalTask = "multisig_approvals"

	// actors raw
	ActorStatesRawTask = "actor_states"
	ActorRawTask       = "actors"

	// rewards
	ChainRewardsTask = "chain_rewards"

	// init
	IdAddressTask = "id_addresses"

	// power
	ChainPowersTask     = "chain_powers"
	PowerActorClaimTask = "power_actor_claims"

	// market
	MarketDealProposalsTask = "market_deal_proposals"
	MarketDealStatesTask    = "market_deal_states"

	// miner
	MinerCurrentDeadlineInfoTask = "miner_current_deadline_infos"
	MinerFeeDebtTask             = "miner_fee_debts"
	MinerInfoTask                = "miner_infos"
	MinerLockedFundsTask         = "miner_locked_funds"
	MinerPreCommitInfoTask       = "miner_pre_commit_infos"
	MinerSectorDealTask          = "miner_sector_deals"
	MinerSectorEventsTask        = "miner_sector_events"
	MinerSectorInfoTask          = "miner_sector_infos"
	MinerSectorPoStTask          = "miner_sector_posts"

	// multisig
	MultiSigTransactionTask = "multisig_transactions"

	// verified registry
	VerifiedRegistryClientTask   = "verified_registry_verified_clients"
	VerifiedRegistryVerifierTask = "verified_registry_verifiers"
)

var AllTasks = []string{
	// blocks
	BlocksTask,
	ChainConsensusTask,
	ChainEconomicsTask,
	DerivedGasOutputsTask,
	MessageGasEconomyTask,
	MessagesTask,
	BlockMessagesTask,
	ParsedMessageTask,
	InternalMessagesTask,
	InternalParsedMessagesTask,
	ReceiptTask,
	MultiSigApprovalTask,
	ActorStatesRawTask,
	ActorRawTask,
	ChainRewardsTask,
	IdAddressTask,
	ChainPowersTask,
	PowerActorClaimTask,
	MarketDealProposalsTask,
	MarketDealStatesTask,
	MinerCurrentDeadlineInfoTask,
	MinerFeeDebtTask,
	MinerInfoTask,
	MinerLockedFundsTask,
	MinerPreCommitInfoTask,
	MinerSectorDealTask,
	MinerSectorEventsTask,
	MinerSectorInfoTask,
	MinerSectorPoStTask,
	MultiSigTransactionTask,
	VerifiedRegistryClientTask,
	VerifiedRegistryVerifierTask,
}
