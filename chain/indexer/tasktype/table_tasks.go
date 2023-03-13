// Code generate by: `make tasks-gen`. DO NOT EDIT.
package tasktype

const (
	BlockHeader                    = "block_header"
	BlockParent                    = "block_parent"
	DrandBlockEntrie               = "drand_block_entrie"
	DataCapBalance                 = "data_cap_balance"
	MinerBeneficiary               = "miner_beneficiary"
	MinerSectorDeal                = "miner_sector_deal"
	MinerSectorInfoV7              = "miner_sector_infos_v7"
	MinerSectorInfoV1_6            = "miner_sector_infos"
	MinerSectorPost                = "miner_sector_post"
	MinerPreCommitInfo             = "miner_pre_commit_info"
	MinerSectorEvent               = "miner_sector_event"
	MinerCurrentDeadlineInfo       = "miner_current_deadline_info"
	MinerFeeDebt                   = "miner_fee_debt"
	MinerLockedFund                = "miner_locked_fund"
	MinerInfo                      = "miner_info"
	MarketDealProposal             = "market_deal_proposal"
	MarketDealState                = "market_deal_state"
	Message                        = "message"
	BlockMessage                   = "block_message"
	Receipt                        = "receipt"
	MessageGasEconomy              = "message_gas_economy"
	ParsedMessage                  = "parsed_message"
	InternalMessage                = "internal_messages"
	InternalParsedMessage          = "internal_parsed_messages"
	VMMessage                      = "vm_messages"
	ActorEvent                     = "actor_events"
	MessageParam                   = "message_params"
	ReceiptReturn                  = "receipt_returns"
	MultisigTransaction            = "multisig_transaction"
	ChainPower                     = "chain_power"
	PowerActorClaim                = "power_actor_claim"
	ChainReward                    = "chain_reward"
	Actor                          = "actor"
	ActorCodes                     = "actor_codes"
	ActorState                     = "actor_state"
	IDAddress                      = "id_addresses"
	GasOutputs                     = "derived_gas_outputs"
	ChainEconomics                 = "chain_economics"
	ChainConsensus                 = "chain_consensus"
	MultisigApproval               = "multisig_approvals"
	VerifiedRegistryVerifier       = "verified_registry_verifier"
	VerifiedRegistryVerifiedClient = "verified_registry_verified_client"
	VerifiedRegistryClaim          = "verified_registry_claim"
	FEVMActorStats                 = "fevm_actor_stats"
)

var AllTableTasks = []string{
	BlockHeader,
	BlockParent,
	DrandBlockEntrie,
	DataCapBalance,
	MinerBeneficiary,
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
	MarketDealProposal,
	MarketDealState,
	Message,
	BlockMessage,
	Receipt,
	MessageGasEconomy,
	ParsedMessage,
	InternalMessage,
	InternalParsedMessage,
	VMMessage,
	ActorEvent,
	MessageParam,
	ReceiptReturn,
	MultisigTransaction,
	ChainPower,
	PowerActorClaim,
	ChainReward,
	Actor,
	ActorCodes,
	ActorState,
	IDAddress,
	GasOutputs,
	ChainEconomics,
	ChainConsensus,
	MultisigApproval,
	VerifiedRegistryVerifier,
	VerifiedRegistryVerifiedClient,
	VerifiedRegistryClaim,
	FEVMActorStats,
}

var TableLookup = map[string]struct{}{
	BlockHeader:                    {},
	BlockParent:                    {},
	DrandBlockEntrie:               {},
	DataCapBalance:                 {},
	MinerBeneficiary:               {},
	MinerSectorDeal:                {},
	MinerSectorInfoV7:              {},
	MinerSectorInfoV1_6:            {},
	MinerSectorPost:                {},
	MinerPreCommitInfo:             {},
	MinerSectorEvent:               {},
	MinerCurrentDeadlineInfo:       {},
	MinerFeeDebt:                   {},
	MinerLockedFund:                {},
	MinerInfo:                      {},
	MarketDealProposal:             {},
	MarketDealState:                {},
	Message:                        {},
	BlockMessage:                   {},
	Receipt:                        {},
	MessageGasEconomy:              {},
	ParsedMessage:                  {},
	InternalMessage:                {},
	InternalParsedMessage:          {},
	VMMessage:                      {},
	ActorEvent:                     {},
	MessageParam:                   {},
	ReceiptReturn:                  {},
	MultisigTransaction:            {},
	ChainPower:                     {},
	PowerActorClaim:                {},
	ChainReward:                    {},
	Actor:                          {},
	ActorCodes:                     {},
	ActorState:                     {},
	IDAddress:                      {},
	GasOutputs:                     {},
	ChainEconomics:                 {},
	ChainConsensus:                 {},
	MultisigApproval:               {},
	VerifiedRegistryVerifier:       {},
	VerifiedRegistryVerifiedClient: {},
	VerifiedRegistryClaim:          {},
	FEVMActorStats:                 {},
}

var TableComment = map[string]string{
	BlockHeader:                    ``,
	BlockParent:                    ``,
	DrandBlockEntrie:               `DrandBlockEntrie contains Drand randomness round numbers used in each block.`,
	DataCapBalance:                 ``,
	MinerBeneficiary:               ``,
	MinerSectorDeal:                ``,
	MinerSectorInfoV7:              `MinerSectorInfoV7 is the default model exported from the miner actor extractor. the table is returned iff the miner actor code is greater than or equal to v7. The table receives a new name since we cannot rename the miner_sector_info table, else we will break backfill.`,
	MinerSectorInfoV1_6:            `MinerSectorInfoV1_6 is exported from the miner actor iff the actor code is less than v7. The table keeps its original name since that's a requirement to support lily backfills`,
	MinerSectorPost:                ``,
	MinerPreCommitInfo:             ``,
	MinerSectorEvent:               ``,
	MinerCurrentDeadlineInfo:       ``,
	MinerFeeDebt:                   ``,
	MinerLockedFund:                ``,
	MinerInfo:                      ``,
	MarketDealProposal:             `MarketDealProposal contains all storage deal states with latest values applied to end_epoch when updates are detected on-chain.`,
	MarketDealState:                ``,
	Message:                        ``,
	BlockMessage:                   ``,
	Receipt:                        ``,
	MessageGasEconomy:              ``,
	ParsedMessage:                  ``,
	InternalMessage:                ``,
	InternalParsedMessage:          ``,
	VMMessage:                      ``,
	ActorEvent:                     ``,
	MessageParam:                   ``,
	ReceiptReturn:                  ``,
	MultisigTransaction:            ``,
	ChainPower:                     ``,
	PowerActorClaim:                ``,
	ChainReward:                    ``,
	Actor:                          `Actor on chain that were added or updated at an epoch. Associates the actor's state root CID (head) with the chain state root CID from which it decends. Includes account ID nonce and balance at each state.`,
	ActorCodes:                     `A mapping of a builtin actor's' CID to a human friendly name.`,
	ActorState:                     `ActorState that were changed at an epoch. Associates actors states as single-level trees with CIDs pointing to complete state tree with the root CID (head) for that actor’s state.`,
	IDAddress:                      `IDAddress contains a mapping of ID addresses to robust addresses from the init actor’s state.`,
	GasOutputs:                     ``,
	ChainEconomics:                 ``,
	ChainConsensus:                 ``,
	MultisigApproval:               ``,
	VerifiedRegistryVerifier:       ``,
	VerifiedRegistryVerifiedClient: ``,
	VerifiedRegistryClaim:          ``,
	FEVMActorStats:                 ``,
}

var TableFieldComments = map[string]map[string]string{
	BlockHeader: {},
	BlockParent: {},
	DrandBlockEntrie: {
		"Block": "Block is the CID of the block.",
		"Round": "Round is the round number of randomness used.",
	},
	DataCapBalance:   {},
	MinerBeneficiary: {},
	MinerSectorDeal:  {},
	MinerSectorInfoV7: {
		"SectorKeyCID": "added in specs-actors v7, will be null for all sectors and only gets set on the first ReplicaUpdate",
	},
	MinerSectorInfoV1_6:      {},
	MinerSectorPost:          {},
	MinerPreCommitInfo:       {},
	MinerSectorEvent:         {},
	MinerCurrentDeadlineInfo: {},
	MinerFeeDebt:             {},
	MinerLockedFund:          {},
	MinerInfo:                {},
	MarketDealProposal: {
		"ClientCollateral":     "The amount of FIL (in attoFIL) the client has pledged as collateral.",
		"ClientID":             "Address of the actor proposing the deal.",
		"DealID":               "Identifier for the deal.",
		"EndEpoch":             "The epoch at which this deal with end.",
		"Height":               "Epoch at which this deal proposal was added or changed.",
		"IsString":             "When true Label contains a valid UTF-8 string encoded in base64. When false Label contains raw bytes encoded in base64. Related to FIP: https://github.com/filecoin-project/FIPs/blob/master/FIPS/fip-0027.md",
		"IsVerified":           "Deal is with a verified provider.",
		"Label":                "An arbitrary client chosen label to apply to the deal. The value is base64 encoded before persisting.",
		"PaddedPieceSize":      "The piece size in bytes with padding.",
		"PieceCID":             "CID of a sector piece. A Piece is an object that represents a whole or part of a File.",
		"ProviderCollateral":   "The amount of FIL (in attoFIL) the provider has pledged as collateral. The Provider deal collateral is only slashed when a sector is terminated before the deal expires.",
		"ProviderID":           "Address of the actor providing the services.",
		"StartEpoch":           "The epoch at which this deal with begin. Storage deal must appear in a sealed (proven) sector no later than start_epoch, otherwise it is invalid.",
		"StateRoot":            "CID of the parent state root for this deal.",
		"StoragePricePerEpoch": "The amount of FIL (in attoFIL) that will be transferred from the client to the provider every epoch this deal is active for.",
		"UnpaddedPieceSize":    "The piece size in bytes without padding.",
	},
	MarketDealState:       {},
	Message:               {},
	BlockMessage:          {},
	Receipt:               {},
	MessageGasEconomy:     {},
	ParsedMessage:         {},
	InternalMessage:       {},
	InternalParsedMessage: {},
	VMMessage: {
		"ActorCode": "ActorCode of To (receiver).",
		"Cid":       "Cid of the message.",
		"ExitCode":  "ExitCode of message execution.",
		"From":      "From sender of message.",
		"GasUsed":   "GasUsed by message.",
		"Height":    "Height message was executed at.",
		"Index":     "Index indicating the order of the messages execution.",
		"Method":    "Method called on To (receiver).",
		"Params":    "Params contained in message.",
		"Returns":   "Returns value of message receipt.",
		"Source":    "On-chain message triggering the message.",
		"StateRoot": "StateRoot message was applied to.",
		"To":        "To receiver of message.",
		"Value":     "Value attoFIL contained in message.",
	},
	ActorEvent:    {},
	MessageParam:  {},
	ReceiptReturn: {},
	MultisigTransaction: {
		"To": "Transaction State",
	},
	ChainPower:      {},
	PowerActorClaim: {},
	ChainReward:     {},
	Actor: {
		"Balance":   "Balance of Actor in attoFIL.",
		"Code":      "Human-readable identifier for the type of the actor.",
		"Head":      "CID of the root of the state tree for the actor.",
		"Height":    "Epoch when this actor was created or updated.",
		"ID":        "ID Actor address.",
		"Nonce":     "The next Actor nonce that is expected to appear on chain.",
		"StateRoot": "CID of the state root when this actor was created or changed.",
	},
	ActorState: {
		"Code":   "CID identifier for the type of the actor.",
		"Head":   "CID of the root of the state tree for the actor.",
		"Height": "Epoch when this actor was created or updated.",
		"State":  "Top level of state data as json.",
	},
	IDAddress: {
		"Address":   "Robust address",
		"Height":    "Epoch when this address mapping was created or updated.",
		"ID":        "ID address",
		"StateRoot": "StateRoot when this address mapping was created or updated.",
	},
	GasOutputs:                     {},
	ChainEconomics:                 {},
	ChainConsensus:                 {},
	MultisigApproval:               {},
	VerifiedRegistryVerifier:       {},
	VerifiedRegistryVerifiedClient: {},
	VerifiedRegistryClaim:          {},
	FEVMActorStats:                 {},
}
