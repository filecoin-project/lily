package migrations

import "github.com/go-pg/migrations/v8"

// Schema version 27 adds documentation in the form of comments to existing schema tables and their columns.

func init() {
	up := batch(`
	COMMENT ON TABLE actor_states IS 'Actor states that were changed at an epoch. Associates actors states as single-level trees with CIDs pointing to complete state tree with the root CID (head) for that actor''s state.';
	COMMENT ON COLUMN actor_states.code IS 'CID identifier for the type of the actor.';
	COMMENT ON COLUMN actor_states.head IS 'CID of the root of the state tree for the actor.';
	COMMENT ON COLUMN actor_states.height IS 'Epoch when this state change happened.';
	COMMENT ON COLUMN actor_states.state IS 'Top level of state data.';

	COMMENT ON TABLE actors IS 'Actors on chain that were added or updated at an epoch. Associates the actor''s state root CID (head) with the chain state root CID from which it decends. Includes account ID nonce and balance at each state.';
	COMMENT ON COLUMN actors.balance IS 'Actor balance in attoFIL.';
	COMMENT ON COLUMN actors.code IS 'Human readable identifier for the type of the actor.';
	COMMENT ON COLUMN actors.head IS 'CID of the root of the state tree for the actor.';
	COMMENT ON COLUMN actors.height IS 'Epoch when this actor was created or updated.';
	COMMENT ON COLUMN actors.id IS 'Actor address.';
	COMMENT ON COLUMN actors.nonce IS 'The next actor nonce that is expected to appear on chain.';
	COMMENT ON COLUMN actors.state_root IS 'CID of the state root.';

	COMMENT ON TABLE block_headers IS 'Blocks included in tipsets at an epoch.';
	COMMENT ON COLUMN block_headers.cid IS 'CID of the block.';
	COMMENT ON COLUMN block_headers.fork_signaling IS 'Flag used as part of signaling forks.';
	COMMENT ON COLUMN block_headers.height IS 'Epoch when this block was mined.';
	COMMENT ON COLUMN block_headers.miner IS 'Address of the miner who mined this block.';
	COMMENT ON COLUMN block_headers.parent_base_fee IS 'The base fee after executing the parent tipset.';
	COMMENT ON COLUMN block_headers.parent_state_root IS 'CID of the block''s parent state root.';
	COMMENT ON COLUMN block_headers.parent_weight IS 'Aggregate chain weight of the block''s parent set.';
	COMMENT ON COLUMN block_headers.timestamp IS 'Time the block was mined in Unix time, the number of seconds elapsed since January 1, 1970 UTC.';
	COMMENT ON COLUMN block_headers.win_count IS 'Number of reward units won in this block.';

	COMMENT ON TABLE block_messages IS 'Message CIDs and the Blocks CID which contain them.';
	COMMENT ON COLUMN block_messages.block IS 'CID of the block that contains the message.';
	COMMENT ON COLUMN block_messages.height IS 'Epoch when the block was mined.';
	COMMENT ON COLUMN block_messages.message IS 'CID of a message in the block.';

	COMMENT ON TABLE block_parents IS 'Block CIDs to many parent Block CIDs.';
	COMMENT ON COLUMN block_parents.block IS 'CID of the block.';
	COMMENT ON COLUMN block_parents.height IS 'Epoch when the block was mined.';
	COMMENT ON COLUMN block_parents.parent IS 'CID of the parent block.';

	COMMENT ON TABLE chain_economics IS 'Economic summaries per state root CID.';
	COMMENT ON COLUMN chain_economics.burnt_fil IS 'Total FIL (attoFIL) burned as part of penalties and on-chain computations.';
	COMMENT ON COLUMN chain_economics.circulating_fil IS 'The amount of FIL (attoFIL) circulating and tradeable in the economy. The basis for Market Cap calculations.';
	COMMENT ON COLUMN chain_economics.locked_fil IS 'The amount of FIL (attoFIL) locked as part of mining, deals, and other mechanisms.';
	COMMENT ON COLUMN chain_economics.mined_fil IS 'The amount of FIL (attoFIL) that has been mined by storage miners.';
	COMMENT ON COLUMN chain_economics.parent_state_root IS 'CID of the parent state root.';
	COMMENT ON COLUMN chain_economics.vested_fil IS 'Total amount of FIL (attoFIL) that is vested from genesis allocation.';

	COMMENT ON TABLE chain_powers IS 'Power summaries from the Power actor.';
	COMMENT ON COLUMN chain_powers.height IS 'Epoch this power summary applies to.';
	COMMENT ON COLUMN chain_powers.miner_count IS 'Total number of miners.';
	COMMENT ON COLUMN chain_powers.participating_miner_count IS 'Total number of miners with power above the minimum miner threshold.';
	COMMENT ON COLUMN chain_powers.qa_smoothed_position_estimate IS 'Total power smoothed position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format.';
	COMMENT ON COLUMN chain_powers.qa_smoothed_velocity_estimate IS 'Total power smoothed velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format.';
	COMMENT ON COLUMN chain_powers.state_root IS 'CID of the parent state root.';
	COMMENT ON COLUMN chain_powers.total_pledge_collateral IS 'Total locked FIL (attoFIL) miners have pledged as collateral in order to participate in the economy.';
	COMMENT ON COLUMN chain_powers.total_qa_bytes_committed IS 'Total provably committed, quality adjusted storage power in bytes.';
	COMMENT ON COLUMN chain_powers.total_qa_bytes_power IS 'Total quality adjusted storage power in bytes in the network.';
	COMMENT ON COLUMN chain_powers.total_raw_bytes_committed IS 'Total provably committed storage power in bytes.';
	COMMENT ON COLUMN chain_powers.total_raw_bytes_power IS 'Total storage power in bytes in the network.';

	COMMENT ON TABLE chain_rewards IS 'Reward summaries from the Reward actor.';
	COMMENT ON COLUMN chain_rewards.cum_sum_baseline IS 'Target that CumsumRealized needs to reach for EffectiveNetworkTime to increase. It is measured in byte-epochs (space * time) representing power committed to the network for some duration.';
	COMMENT ON COLUMN chain_rewards.cum_sum_realized IS 'Cumulative sum of network power capped by BaselinePower(epoch). It is measured in byte-epochs (space * time) representing power committed to the network for some duration.';
	COMMENT ON COLUMN chain_rewards.effective_baseline_power IS 'The baseline power (in bytes) at the EffectiveNetworkTime epoch.';
	COMMENT ON COLUMN chain_rewards.effective_network_time IS 'Ceiling of real effective network time "theta" based on CumsumBaselinePower(theta) == CumsumRealizedPower. Theta captures the notion of how much the network has progressed in its baseline and in advancing network time.';
	COMMENT ON COLUMN chain_rewards.height IS 'Epoch this rewards summary applies to.';
	COMMENT ON COLUMN chain_rewards.new_baseline_power IS 'The baseline power (in bytes) the network is targeting.';
	COMMENT ON COLUMN chain_rewards.new_reward IS 'The reward to be paid in per WinCount to block producers. The actual reward total paid out depends on the number of winners in any round. This value is recomputed every non-null epoch and used in the next non-null epoch.';
	COMMENT ON COLUMN chain_rewards.new_reward_smoothed_position_estimate IS 'Smoothed reward position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format.';
	COMMENT ON COLUMN chain_rewards.new_reward_smoothed_velocity_estimate IS 'Smoothed reward velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format.';
	COMMENT ON COLUMN chain_rewards.state_root IS 'CID of the parent state root.';
	COMMENT ON COLUMN chain_rewards.total_mined_reward IS 'The total FIL (attoFIL) awarded to block miners.';

	COMMENT ON TABLE derived_gas_outputs IS 'Derived gas costs resulting from execution of a message in the VM.';
	COMMENT ON COLUMN derived_gas_outputs.actor_name IS 'Human readable identifier for the type of the actor.';
	COMMENT ON COLUMN derived_gas_outputs.base_fee_burn IS 'The amount of FIL (in attoFIL) to burn as a result of the base fee. It is parent_base_fee (or gas_fee_cap if smaller) multiplied by gas_used. Note: successfull window PoSt messages are not charged this burn.';
	COMMENT ON COLUMN derived_gas_outputs.cid IS 'CID of the message.';
	COMMENT ON COLUMN derived_gas_outputs.exit_code IS 'The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific.';
	COMMENT ON COLUMN derived_gas_outputs.from IS 'Address of actor that sent the message.';
	COMMENT ON COLUMN derived_gas_outputs.gas_burned IS 'The overestimated units of gas to burn. It is a portion of the difference between gas_limit and gas_used.';
	COMMENT ON COLUMN derived_gas_outputs.gas_fee_cap IS 'The maximum price that the message sender is willing to pay per unit of gas.';
	COMMENT ON COLUMN derived_gas_outputs.gas_limit IS 'A hard limit on the amount of gas (i.e., number of units of gas) that a message’s execution should be allowed to consume on chain. It is measured in units of gas.';
	COMMENT ON COLUMN derived_gas_outputs.gas_premium IS 'The price per unit of gas (measured in attoFIL/gas) that the message sender is willing to pay (on top of the BaseFee) to "tip" the miner that will include this message in a block.';
	COMMENT ON COLUMN derived_gas_outputs.gas_refund IS 'The overestimated units of gas to refund. It is a portion of the difference between gas_limit and gas_used.';
	COMMENT ON COLUMN derived_gas_outputs.gas_used IS 'A measure of the amount of resources (or units of gas) consumed, in order to execute a message.';
	COMMENT ON COLUMN derived_gas_outputs.height IS 'Epoch this message was executed at.';
	COMMENT ON COLUMN derived_gas_outputs.method IS 'The method number to invoke. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';
	COMMENT ON COLUMN derived_gas_outputs.miner_penalty IS 'Any penalty fees (in attoFIL) the miner incured while executing the message.';
	COMMENT ON COLUMN derived_gas_outputs.miner_tip IS 'The amount of FIL (in attoFIL) the miner receives for executing the message. Typically it is gas_premium * gas_limit but may be lower if the total fees exceed the gas_fee_cap.';
	COMMENT ON COLUMN derived_gas_outputs.nonce IS 'The message nonce, which protects against duplicate messages and multiple messages with the same values.';
	COMMENT ON COLUMN derived_gas_outputs.over_estimation_burn IS 'The fee to pay (in attoFIL) for overestimating the gas used to execute a message. The overestimated gas to burn (gas_burned) is a portion of the difference between gas_limit and gas_used. The over_estimation_burn value is gas_burned * parent_base_fee.';
	COMMENT ON COLUMN derived_gas_outputs.parent_base_fee IS 'The set price per unit of gas (measured in attoFIL/gas unit) to be burned (sent to an unrecoverable address) for every message execution.';
	COMMENT ON COLUMN derived_gas_outputs.refund IS 'The amount of FIL (in attoFIL) to refund to the message sender after base fee, miner tip and overestimation amounts have been deducted.';
	COMMENT ON COLUMN derived_gas_outputs.size_bytes IS 'Size in bytes of the serialized message.';
	COMMENT ON COLUMN derived_gas_outputs.state_root IS 'CID of the parent state root.';
	COMMENT ON COLUMN derived_gas_outputs.to IS 'Address of actor that received the message.';
	COMMENT ON COLUMN derived_gas_outputs.value IS 'The FIL value transferred (attoFIL) to the message receiver.';

	COMMENT ON TABLE drand_block_entries IS 'Drand randomness round numbers used in each block.';
	COMMENT ON COLUMN drand_block_entries.block IS 'CID of the block.';
	COMMENT ON COLUMN drand_block_entries.round IS 'The round number of the randomness used.';

	COMMENT ON TABLE id_addresses IS 'Mapping of IDs to robust addresses from the init actor''s state.';
	COMMENT ON COLUMN id_addresses.address IS 'Robust address of the actor.';
	COMMENT ON COLUMN id_addresses.id IS 'ID of the actor.';
	COMMENT ON COLUMN id_addresses.state_root IS 'CID of the parent state root at which this address mapping was added.';

	COMMENT ON TABLE market_deal_proposals IS 'All storage deal states with latest values applied to end_epoch when updates are detected on-chain.';
	COMMENT ON COLUMN market_deal_proposals.client_collateral IS 'The amount of FIL (in attoFIL) the client has pledged as collateral.';
	COMMENT ON COLUMN market_deal_proposals.client_id IS 'Address of the actor proposing the deal.';
	COMMENT ON COLUMN market_deal_proposals.deal_id IS 'Identifier for the deal.';
	COMMENT ON COLUMN market_deal_proposals.end_epoch IS 'The epoch at which this deal with end.';
	COMMENT ON COLUMN market_deal_proposals.height IS 'Epoch at which this deal proposal was added or changed.';
	COMMENT ON COLUMN market_deal_proposals.is_verified IS 'Deal is with a verified provider.';
	COMMENT ON COLUMN market_deal_proposals.label IS 'An arbitrary client chosen label to apply to the deal.';
	COMMENT ON COLUMN market_deal_proposals.padded_piece_size IS 'The piece size in bytes with padding.';
	COMMENT ON COLUMN market_deal_proposals.piece_cid IS 'CID of a sector piece. A Piece is an object that represents a whole or part of a File.';
	COMMENT ON COLUMN market_deal_proposals.provider_collateral IS 'The amount of FIL (in attoFIL) the provider has pledged as collateral. The Provider deal collateral is only slashed when a sector is terminated before the deal expires.';
	COMMENT ON COLUMN market_deal_proposals.provider_id IS 'Address of the actor providing the services.';
	COMMENT ON COLUMN market_deal_proposals.start_epoch IS 'The epoch at which this deal with begin. Storage deal must appear in a sealed (proven) sector no later than start_epoch, otherwise it is invalid.';
	COMMENT ON COLUMN market_deal_proposals.state_root IS 'CID of the parent state root for this deal.';
	COMMENT ON COLUMN market_deal_proposals.storage_price_per_epoch IS 'The amount of FIL (in attoFIL) that will be transferred from the client to the provider every epoch this deal is active for.';
	COMMENT ON COLUMN market_deal_proposals.unpadded_piece_size IS 'The piece size in bytes without padding.';

	COMMENT ON TABLE market_deal_states IS 'All storage deal state transitions detected on-chain.';
	COMMENT ON COLUMN market_deal_states.deal_id IS 'Identifier for the deal.';
	COMMENT ON COLUMN market_deal_states.height IS 'Epoch at which this deal was added or changed.';
	COMMENT ON COLUMN market_deal_states.last_update_epoch IS 'Epoch this deal was last updated at. -1 if deal state never updated.';
	COMMENT ON COLUMN market_deal_states.sector_start_epoch IS 'Epoch this deal was included in a proven sector. -1 if not yet included in proven sector.';
	COMMENT ON COLUMN market_deal_states.slash_epoch IS 'Epoch this deal was slashed at. -1 if deal was never slashed.';
	COMMENT ON COLUMN market_deal_states.state_root IS 'CID of the parent state root for this deal.';

	COMMENT ON TABLE message_gas_economy IS 'Gas economics for all messages in all blocks at each epoch.';
	COMMENT ON COLUMN message_gas_economy.base_fee IS 'The set price per unit of gas (measured in attoFIL/gas unit) to be burned (sent to an unrecoverable address) for every message execution.';
	COMMENT ON COLUMN message_gas_economy.base_fee_change_log IS 'The logarithm of the change between new and old base fee.';
	COMMENT ON COLUMN message_gas_economy.gas_capacity_ratio IS 'The gas_limit_unique_total / target gas limit total for all blocks.';
	COMMENT ON COLUMN message_gas_economy.gas_fill_ratio IS 'The gas_limit_total / target gas limit total for all blocks.';
	COMMENT ON COLUMN message_gas_economy.gas_limit_total IS 'The sum of all the gas limits.';
	COMMENT ON COLUMN message_gas_economy.gas_limit_unique_total IS 'The sum of all the gas limits of unique messages.';
	COMMENT ON COLUMN message_gas_economy.gas_waste_ratio IS '(gas_limit_total - gas_limit_unique_total) / target gas limit total for all blocks.';
	COMMENT ON COLUMN message_gas_economy.height IS 'Epoch these economics apply to.';
	COMMENT ON COLUMN message_gas_economy.state_root IS 'CID of the parent state root at this epoch.';

	COMMENT ON TABLE messages IS 'Validated on-chain messages by their CID and their metadata.';
	COMMENT ON COLUMN messages.cid IS 'CID of the message.';
	COMMENT ON COLUMN messages.from IS 'Address of the actor that sent the message.';
	COMMENT ON COLUMN messages.gas_fee_cap IS 'The maximum price that the message sender is willing to pay per unit of gas.';
	COMMENT ON COLUMN messages.gas_premium IS 'The price per unit of gas (measured in attoFIL/gas) that the message sender is willing to pay (on top of the BaseFee) to "tip" the miner that will include this message in a block.';
	COMMENT ON COLUMN messages.height IS 'Epoch this message was executed at.';
	COMMENT ON COLUMN messages.method IS 'The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';
	COMMENT ON COLUMN messages.nonce IS 'The message nonce, which protects against duplicate messages and multiple messages with the same values.';
	COMMENT ON COLUMN messages.size_bytes IS 'Size of the serialized message in bytes.';
	COMMENT ON COLUMN messages.to IS 'Address of the actor that received the message.';
	COMMENT ON COLUMN messages.value IS 'Amount of FIL (in attoFIL) transferred by this message.';

	COMMENT ON TABLE miner_current_deadline_infos IS 'Deadline refers to the window during which proofs may be submitted.';
	COMMENT ON COLUMN miner_current_deadline_infos.challenge IS 'Epoch at which to sample the chain for challenge (< Open).';
	COMMENT ON COLUMN miner_current_deadline_infos.close IS 'First epoch from which a proof may no longer be submitted (>= Open).';
	COMMENT ON COLUMN miner_current_deadline_infos.deadline_index IS 'A deadline index, in [0..d.WPoStProvingPeriodDeadlines) unless period elapsed.';
	COMMENT ON COLUMN miner_current_deadline_infos.fault_cutoff IS 'First epoch at which a fault declaration is rejected (< Open).';
	COMMENT ON COLUMN miner_current_deadline_infos.height IS 'Epoch at which this info was calculated.';
	COMMENT ON COLUMN miner_current_deadline_infos.miner_id IS 'Address of the miner this info relates to.';
	COMMENT ON COLUMN miner_current_deadline_infos.open IS 'First epoch from which a proof may be submitted (>= CurrentEpoch).';
	COMMENT ON COLUMN miner_current_deadline_infos.period_start IS 'First epoch of the proving period (<= CurrentEpoch).';
	COMMENT ON COLUMN miner_current_deadline_infos.state_root IS 'CID of the parent state root at this epoch.';

	COMMENT ON TABLE miner_fee_debts IS 'Miner debts per epoch from unpaid fees.';
	COMMENT ON COLUMN miner_fee_debts.fee_debt IS 'Absolute value of debt this miner owes from unpaid fees in attoFIL.';
	COMMENT ON COLUMN miner_fee_debts.height IS 'Epoch at which this debt applies.';
	COMMENT ON COLUMN miner_fee_debts.miner_id IS 'Address of the miner that owes fees.';
	COMMENT ON COLUMN miner_fee_debts.state_root IS 'CID of the parent state root at this epoch.';

	COMMENT ON TABLE miner_infos IS 'Miner Account IDs for all associated addresses plus peer ID. See https://docs.filecoin.io/mine/lotus/miner-addresses/ for more information.';
	COMMENT ON COLUMN miner_infos.consensus_faulted_elapsed IS 'The next epoch this miner is eligible for certain permissioned actor methods and winning block elections as a result of being reported for a consensus fault.';
	COMMENT ON COLUMN miner_infos.control_addresses IS 'JSON array of control addresses. Control addresses are used to submit WindowPoSts proofs to the chain. WindowPoSt is the mechanism through which storage is verified in Filecoin and is required by miners to submit proofs for all sectors every 24 hours. Those proofs are submitted as messages to the blockchain and therefore need to pay the respective fees.';
	COMMENT ON COLUMN miner_infos.height IS 'Epoch at which this miner info was added/changed.';
	COMMENT ON COLUMN miner_infos.miner_id IS 'Address of miner this info applies to.';
	COMMENT ON COLUMN miner_infos.multi_addresses IS 'JSON array of multiaddrs at which this miner can be reached.';
	COMMENT ON COLUMN miner_infos.new_worker IS 'Address of a new worker address that will become effective at worker_change_epoch.';
	COMMENT ON COLUMN miner_infos.owner_id IS 'Address of actor designated as the owner. The owner address is the address that created the miner, paid the collateral, and has block rewards paid out to it.';
	COMMENT ON COLUMN miner_infos.peer_id IS 'Current libp2p Peer ID of the miner.';
	COMMENT ON COLUMN miner_infos.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN miner_infos.worker_id IS 'Address of actor designated as the worker. The worker is responsible for doing all of the work, submitting proofs, committing new sectors, and all other day to day activities.';
	COMMENT ON COLUMN miner_infos.worker_change_epoch IS 'Epoch at which a new_worker address will become effective.';

	COMMENT ON TABLE miner_locked_funds IS 'Details of Miner funds locked and unavailable for use.';
	COMMENT ON COLUMN miner_locked_funds.height IS 'Epoch at which these details were added/changed.';
	COMMENT ON COLUMN miner_locked_funds.initial_pledge IS 'Amount of FIL (in attoFIL) locked due to it being pledged as collateral. When a Miner ProveCommits a Sector, they must supply an "initial pledge" for the Sector, which acts as collateral. If the Sector is terminated, this deposit is removed and burned along with rewards earned by this sector up to a limit.';
	COMMENT ON COLUMN miner_locked_funds.locked_funds IS 'Amount of FIL (in attoFIL) locked due to vesting. When a Miner receives tokens from block rewards, the tokens are locked and added to the Miner''s vesting table to be unlocked linearly over some future epochs.';
	COMMENT ON COLUMN miner_locked_funds.miner_id IS 'Address of the miner these details apply to.';
	COMMENT ON COLUMN miner_locked_funds.pre_commit_deposits IS 'Amount of FIL (in attoFIL) locked due to it being used as a PreCommit deposit. When a Miner PreCommits a Sector, they must supply a "precommit deposit" for the Sector, which acts as collateral. If the Sector is not ProveCommitted on time, this deposit is removed and burned.';
	COMMENT ON COLUMN miner_locked_funds.state_root IS 'CID of the parent state root at this epoch.';

	COMMENT ON TABLE miner_pre_commit_infos IS 'Information on sector PreCommits.';
	COMMENT ON COLUMN miner_pre_commit_infos.deal_weight IS 'Total space*time of submitted deals.';
	COMMENT ON COLUMN miner_pre_commit_infos.expiration_epoch IS 'Epoch this sector expires.';
	COMMENT ON COLUMN miner_pre_commit_infos.height IS 'Epoch this PreCommit information was added/changed.';
	COMMENT ON COLUMN miner_pre_commit_infos.is_replace_capacity IS 'Whether to replace a "committed capacity" no-deal sector (requires non-empty DealIDs).';
	COMMENT ON COLUMN miner_pre_commit_infos.miner_id IS 'Address of the miner who owns the sector.';
	COMMENT ON COLUMN miner_pre_commit_infos.pre_commit_deposit IS 'Amount of FIL (in attoFIL) used as a PreCommit deposit. If the Sector is not ProveCommitted on time, this deposit is removed and burned.';
	COMMENT ON COLUMN miner_pre_commit_infos.pre_commit_epoch IS 'Epoch this PreCommit was created.';
	COMMENT ON COLUMN miner_pre_commit_infos.replace_sector_deadline IS 'The deadline location of the sector to replace.';
	COMMENT ON COLUMN miner_pre_commit_infos.replace_sector_number IS 'ID of the committed capacity sector to replace.';
	COMMENT ON COLUMN miner_pre_commit_infos.replace_sector_partition IS 'The partition location of the sector to replace.';
	COMMENT ON COLUMN miner_pre_commit_infos.seal_rand_epoch IS 'Seal challenge epoch. Epoch at which randomness should be drawn to tie Proof-of-Replication to a chain.';
	COMMENT ON COLUMN miner_pre_commit_infos.sealed_cid IS 'CID of the sealed sector.';
	COMMENT ON COLUMN miner_pre_commit_infos.sector_id IS 'Numeric identifier for the sector.';
	COMMENT ON COLUMN miner_pre_commit_infos.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN miner_pre_commit_infos.verified_deal_weight IS 'Total space*time of submitted verified deals.';

	COMMENT ON TABLE miner_sector_deals IS 'Mapping of Deal IDs to their respective Miner and Sector IDs.';
	COMMENT ON COLUMN miner_sector_deals.deal_id IS 'Numeric identifier for the deal.';
	COMMENT ON COLUMN miner_sector_deals.height IS 'Epoch at which this deal was added/updated.';
	COMMENT ON COLUMN miner_sector_deals.miner_id IS 'Address of the miner the deal is with.';
	COMMENT ON COLUMN miner_sector_deals.sector_id IS 'Numeric identifier of the sector the deal is for.';

	COMMENT ON TABLE miner_sector_events IS 'Sector events on-chain per Miner/Sector.';
	COMMENT ON COLUMN miner_sector_events.event IS 'Name of the event that occurred.';
	COMMENT ON COLUMN miner_sector_events.height IS 'Epoch at which this event occurred.';
	COMMENT ON COLUMN miner_sector_events.miner_id IS 'Address of the miner who owns the sector.';
	COMMENT ON COLUMN miner_sector_events.sector_id IS 'Numeric identifier of the sector.';
	COMMENT ON COLUMN miner_sector_events.state_root IS 'CID of the parent state root at this epoch.';

	COMMENT ON TABLE miner_sector_infos IS 'Latest state of sectors by Miner.';
	COMMENT ON COLUMN miner_sector_infos.activation_epoch IS 'Epoch during which the sector proof was accepted.';
	COMMENT ON COLUMN miner_sector_infos.deal_weight IS 'Integral of active deals over sector lifetime.';
	COMMENT ON COLUMN miner_sector_infos.expected_day_reward IS 'Expected one day projection of reward for sector computed at activation time (in attoFIL).';
	COMMENT ON COLUMN miner_sector_infos.expected_storage_pledge IS 'Expected twenty day projection of reward for sector computed at activation time (in attoFIL).';
	COMMENT ON COLUMN miner_sector_infos.expiration_epoch IS 'Epoch during which the sector expires.';
	COMMENT ON COLUMN miner_sector_infos.height IS 'Epoch at which this sector info was added/updated.';
	COMMENT ON COLUMN miner_sector_infos.initial_pledge IS 'Pledge collected to commit this sector (in attoFIL).';
	COMMENT ON COLUMN miner_sector_infos.miner_id IS 'Address of the miner who owns the sector.';
	COMMENT ON COLUMN miner_sector_infos.sealed_cid IS 'The root CID of the Sealed Sector’s merkle tree. Also called CommR, or "replica commitment".';
	COMMENT ON COLUMN miner_sector_infos.sector_id IS 'Numeric identifier of the sector.';
	COMMENT ON COLUMN miner_sector_infos.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN miner_sector_infos.verified_deal_weight IS 'Integral of active verified deals over sector lifetime.';

	COMMENT ON TABLE miner_sector_posts IS 'Proof of Spacetime for sectors.';
	COMMENT ON COLUMN miner_sector_posts.height IS 'Epoch at which this PoSt message was executed.';
	COMMENT ON COLUMN miner_sector_posts.miner_id IS 'Address of the miner who owns the sector.';
	COMMENT ON COLUMN miner_sector_posts.post_message_cid IS 'CID of the PoSt message.';
	COMMENT ON COLUMN miner_sector_posts.sector_id IS 'Numeric identifier of the sector.';

	COMMENT ON TABLE multisig_transactions IS 'Details of pending transactions involving multisig actors.';
	COMMENT ON COLUMN multisig_transactions.approved IS 'Addresses of signers who have approved the transaction. 0th entry is the proposer.';
	COMMENT ON COLUMN multisig_transactions.height IS 'Epoch at which this transaction was executed.';
	COMMENT ON COLUMN multisig_transactions.method IS 'The method number to invoke on the recipient if the proposal is approved. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution.';
	COMMENT ON COLUMN multisig_transactions.multisig_id IS 'Address of the multisig actor involved in the transaction.';
	COMMENT ON COLUMN multisig_transactions.params IS 'CBOR encoded bytes of parameters to send to the method that will be invoked if the proposal is approved.';
	COMMENT ON COLUMN multisig_transactions.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN multisig_transactions.to IS 'Address of the recipient who will be sent a message if the proposal is approved.';
	COMMENT ON COLUMN multisig_transactions.transaction_id IS 'Number identifier for the transaction - unique per multisig.';
	COMMENT ON COLUMN multisig_transactions.value IS 'Amount of FIL (in attoFIL) that will be transferred if the proposal is approved.';
`)
	down := batch(`
	COMMENT ON TABLE actor_states IS NULL;
	COMMENT ON COLUMN actor_states.code IS NULL;
	COMMENT ON COLUMN actor_states.head IS NULL;
	COMMENT ON COLUMN actor_states.height IS NULL;
	COMMENT ON COLUMN actor_states.state IS NULL;

	COMMENT ON TABLE actors IS NULL;
	COMMENT ON COLUMN actors.balance IS NULL;
	COMMENT ON COLUMN actors.code IS NULL;
	COMMENT ON COLUMN actors.head IS NULL;
	COMMENT ON COLUMN actors.height IS NULL;
	COMMENT ON COLUMN actors.id IS NULL;
	COMMENT ON COLUMN actors.nonce IS NULL;
	COMMENT ON COLUMN actors.state_root IS NULL;

	COMMENT ON TABLE block_headers IS NULL;
	COMMENT ON COLUMN block_headers.cid IS NULL;
	COMMENT ON COLUMN block_headers.fork_signaling IS NULL;
	COMMENT ON COLUMN block_headers.height IS NULL;
	COMMENT ON COLUMN block_headers.miner IS NULL;
	COMMENT ON COLUMN block_headers.parent_base_fee IS NULL;
	COMMENT ON COLUMN block_headers.parent_state_root IS NULL;
	COMMENT ON COLUMN block_headers.parent_weight IS NULL;
	COMMENT ON COLUMN block_headers.timestamp IS NULL;
	COMMENT ON COLUMN block_headers.win_count IS NULL;

	COMMENT ON TABLE block_messages IS NULL;
	COMMENT ON COLUMN block_messages.block IS NULL;
	COMMENT ON COLUMN block_messages.height IS NULL;
	COMMENT ON COLUMN block_messages.message IS NULL;

	COMMENT ON TABLE block_parents IS NULL;
	COMMENT ON COLUMN block_parents.block IS NULL;
	COMMENT ON COLUMN block_parents.height IS NULL;
	COMMENT ON COLUMN block_parents.parent IS NULL;

	COMMENT ON TABLE chain_economics IS NULL;
	COMMENT ON COLUMN chain_economics.burnt_fil IS NULL;
	COMMENT ON COLUMN chain_economics.circulating_fil IS NULL;
	COMMENT ON COLUMN chain_economics.locked_fil IS NULL;
	COMMENT ON COLUMN chain_economics.mined_fil IS NULL;
	COMMENT ON COLUMN chain_economics.parent_state_root IS NULL;
	COMMENT ON COLUMN chain_economics.vested_fil IS NULL;

	COMMENT ON TABLE chain_powers IS NULL;
	COMMENT ON COLUMN chain_powers.height IS NULL;
	COMMENT ON COLUMN chain_powers.miner_count IS NULL;
	COMMENT ON COLUMN chain_powers.participating_miner_count IS NULL;
	COMMENT ON COLUMN chain_powers.qa_smoothed_position_estimate IS NULL;
	COMMENT ON COLUMN chain_powers.qa_smoothed_velocity_estimate IS NULL;
	COMMENT ON COLUMN chain_powers.state_root IS NULL;
	COMMENT ON COLUMN chain_powers.total_pledge_collateral IS NULL;
	COMMENT ON COLUMN chain_powers.total_qa_bytes_committed IS NULL;
	COMMENT ON COLUMN chain_powers.total_qa_bytes_power IS NULL;
	COMMENT ON COLUMN chain_powers.total_raw_bytes_committed IS NULL;
	COMMENT ON COLUMN chain_powers.total_raw_bytes_power IS NULL;

	COMMENT ON TABLE chain_rewards IS NULL;
	COMMENT ON COLUMN chain_rewards.cum_sum_baseline IS NULL;
	COMMENT ON COLUMN chain_rewards.cum_sum_realized IS NULL;
	COMMENT ON COLUMN chain_rewards.effective_baseline_power IS NULL;
	COMMENT ON COLUMN chain_rewards.effective_network_time IS NULL;
	COMMENT ON COLUMN chain_rewards.height IS NULL;
	COMMENT ON COLUMN chain_rewards.new_baseline_power IS NULL;
	COMMENT ON COLUMN chain_rewards.new_reward IS NULL;
	COMMENT ON COLUMN chain_rewards.new_reward_smoothed_position_estimate IS NULL;
	COMMENT ON COLUMN chain_rewards.new_reward_smoothed_velocity_estimate IS NULL;
	COMMENT ON COLUMN chain_rewards.state_root IS NULL;
	COMMENT ON COLUMN chain_rewards.total_mined_reward IS NULL;

	COMMENT ON TABLE derived_gas_outputs IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.actor_name IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.base_fee_burn IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.cid IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.exit_code IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.from IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.gas_burned IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.gas_fee_cap IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.gas_limit IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.gas_premium IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.gas_refund IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.gas_used IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.height IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.method IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.miner_penalty IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.miner_tip IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.nonce IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.over_estimation_burn IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.parent_base_fee IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.refund IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.size_bytes IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.state_root IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.to IS NULL;
	COMMENT ON COLUMN derived_gas_outputs.value IS NULL;

	COMMENT ON TABLE drand_block_entries IS NULL;
	COMMENT ON COLUMN drand_block_entries.block IS NULL;
	COMMENT ON COLUMN drand_block_entries.round IS NULL;

	COMMENT ON TABLE id_addresses IS NULL;
	COMMENT ON COLUMN id_addresses.address IS NULL;
	COMMENT ON COLUMN id_addresses.id IS NULL;
	COMMENT ON COLUMN id_addresses.state_root IS NULL;

	COMMENT ON TABLE market_deal_proposals IS NULL;
	COMMENT ON COLUMN market_deal_proposals.client_collateral IS NULL;
	COMMENT ON COLUMN market_deal_proposals.client_id IS NULL;
	COMMENT ON COLUMN market_deal_proposals.deal_id IS NULL;
	COMMENT ON COLUMN market_deal_proposals.end_epoch IS NULL;
	COMMENT ON COLUMN market_deal_proposals.height IS NULL;
	COMMENT ON COLUMN market_deal_proposals.is_verified IS NULL;
	COMMENT ON COLUMN market_deal_proposals.label IS NULL;
	COMMENT ON COLUMN market_deal_proposals.padded_piece_size IS NULL;
	COMMENT ON COLUMN market_deal_proposals.piece_cid IS NULL;
	COMMENT ON COLUMN market_deal_proposals.provider_collateral IS NULL;
	COMMENT ON COLUMN market_deal_proposals.provider_id IS NULL;
	COMMENT ON COLUMN market_deal_proposals.start_epoch IS NULL;
	COMMENT ON COLUMN market_deal_proposals.state_root IS NULL;
	COMMENT ON COLUMN market_deal_proposals.storage_price_per_epoch IS NULL;
	COMMENT ON COLUMN market_deal_proposals.unpadded_piece_size IS NULL;

	COMMENT ON TABLE market_deal_states IS NULL;
	COMMENT ON COLUMN market_deal_states.deal_id IS NULL;
	COMMENT ON COLUMN market_deal_states.height IS NULL;
	COMMENT ON COLUMN market_deal_states.last_update_epoch IS NULL;
	COMMENT ON COLUMN market_deal_states.sector_start_epoch IS NULL;
	COMMENT ON COLUMN market_deal_states.slash_epoch IS NULL;
	COMMENT ON COLUMN market_deal_states.state_root IS NULL;

	COMMENT ON TABLE message_gas_economy IS NULL;
	COMMENT ON COLUMN message_gas_economy.base_fee IS NULL;
	COMMENT ON COLUMN message_gas_economy.base_fee_change_log IS NULL;
	COMMENT ON COLUMN message_gas_economy.gas_capacity_ratio IS NULL;
	COMMENT ON COLUMN message_gas_economy.gas_fill_ratio IS NULL;
	COMMENT ON COLUMN message_gas_economy.gas_limit_total IS NULL;
	COMMENT ON COLUMN message_gas_economy.gas_limit_unique_total IS NULL;
	COMMENT ON COLUMN message_gas_economy.gas_waste_ratio IS NULL;
	COMMENT ON COLUMN message_gas_economy.height IS NULL;
	COMMENT ON COLUMN message_gas_economy.state_root IS NULL;

	COMMENT ON TABLE messages IS NULL;
	COMMENT ON COLUMN messages.cid IS NULL;
	COMMENT ON COLUMN messages.from IS NULL;
	COMMENT ON COLUMN messages.gas_fee_cap IS NULL;
	COMMENT ON COLUMN messages.gas_premium IS NULL;
	COMMENT ON COLUMN messages.height IS NULL;
	COMMENT ON COLUMN messages.method IS NULL;
	COMMENT ON COLUMN messages.nonce IS NULL;
	COMMENT ON COLUMN messages.size_bytes IS NULL;
	COMMENT ON COLUMN messages.to IS NULL;
	COMMENT ON COLUMN messages.value IS NULL;

	COMMENT ON TABLE miner_current_deadline_infos IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.challenge IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.close IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.deadline_index IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.fault_cutoff IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.height IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.miner_id IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.open IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.period_start IS NULL;
	COMMENT ON COLUMN miner_current_deadline_infos.state_root IS NULL;

	COMMENT ON TABLE miner_fee_debts IS NULL;
	COMMENT ON COLUMN miner_fee_debts.fee_debt IS NULL;
	COMMENT ON COLUMN miner_fee_debts.height IS NULL;
	COMMENT ON COLUMN miner_fee_debts.miner_id IS NULL;
	COMMENT ON COLUMN miner_fee_debts.state_root IS NULL;

	COMMENT ON TABLE miner_infos IS NULL;
	COMMENT ON COLUMN miner_infos.consensus_faulted_elapsed IS NULL;
	COMMENT ON COLUMN miner_infos.control_addresses IS NULL;
	COMMENT ON COLUMN miner_infos.height IS NULL;
	COMMENT ON COLUMN miner_infos.miner_id IS NULL;
	COMMENT ON COLUMN miner_infos.multi_addresses IS NULL;
	COMMENT ON COLUMN miner_infos.new_worker IS NULL;
	COMMENT ON COLUMN miner_infos.owner_id IS NULL;
	COMMENT ON COLUMN miner_infos.peer_id IS NULL;
	COMMENT ON COLUMN miner_infos.state_root IS NULL;
	COMMENT ON COLUMN miner_infos.worker_id IS NULL;
	COMMENT ON COLUMN miner_infos.worker_change_epoch IS NULL;

	COMMENT ON TABLE miner_locked_funds IS NULL;
	COMMENT ON COLUMN miner_locked_funds.height IS NULL;
	COMMENT ON COLUMN miner_locked_funds.initial_pledge IS NULL;
	COMMENT ON COLUMN miner_locked_funds.locked_funds IS NULL;
	COMMENT ON COLUMN miner_locked_funds.miner_id IS NULL;
	COMMENT ON COLUMN miner_locked_funds.pre_commit_deposits IS NULL;
	COMMENT ON COLUMN miner_locked_funds.state_root IS NULL;

	COMMENT ON TABLE miner_pre_commit_infos IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.deal_weight IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.expiration_epoch IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.height IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.is_replace_capacity IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.miner_id IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.pre_commit_deposit IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.pre_commit_epoch IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.replace_sector_deadline IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.replace_sector_number IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.replace_sector_partition IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.seal_rand_epoch IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.sealed_cid IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.sector_id IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.state_root IS NULL;
	COMMENT ON COLUMN miner_pre_commit_infos.verified_deal_weight IS NULL;

	COMMENT ON TABLE miner_sector_deals IS NULL;
	COMMENT ON COLUMN miner_sector_deals.deal_id IS NULL;
	COMMENT ON COLUMN miner_sector_deals.height IS NULL;
	COMMENT ON COLUMN miner_sector_deals.miner_id IS NULL;
	COMMENT ON COLUMN miner_sector_deals.sector_id IS NULL;

	COMMENT ON TABLE miner_sector_events IS NULL;
	COMMENT ON COLUMN miner_sector_events.event IS NULL;
	COMMENT ON COLUMN miner_sector_events.height IS NULL;
	COMMENT ON COLUMN miner_sector_events.miner_id IS NULL;
	COMMENT ON COLUMN miner_sector_events.sector_id IS NULL;
	COMMENT ON COLUMN miner_sector_events.state_root IS NULL;

	COMMENT ON TABLE miner_sector_infos IS NULL;
	COMMENT ON COLUMN miner_sector_infos.activation_epoch IS NULL;
	COMMENT ON COLUMN miner_sector_infos.deal_weight IS NULL;
	COMMENT ON COLUMN miner_sector_infos.expected_day_reward IS NULL;
	COMMENT ON COLUMN miner_sector_infos.expected_storage_pledge IS NULL;
	COMMENT ON COLUMN miner_sector_infos.expiration_epoch IS NULL;
	COMMENT ON COLUMN miner_sector_infos.height IS NULL;
	COMMENT ON COLUMN miner_sector_infos.initial_pledge IS NULL;
	COMMENT ON COLUMN miner_sector_infos.miner_id IS NULL;
	COMMENT ON COLUMN miner_sector_infos.sealed_cid IS NULL;
	COMMENT ON COLUMN miner_sector_infos.sector_id IS NULL;
	COMMENT ON COLUMN miner_sector_infos.state_root IS NULL;
	COMMENT ON COLUMN miner_sector_infos.verified_deal_weight IS NULL;

	COMMENT ON TABLE miner_sector_posts IS NULL;
	COMMENT ON COLUMN miner_sector_posts.height IS NULL;
	COMMENT ON COLUMN miner_sector_posts.miner_id IS NULL;
	COMMENT ON COLUMN miner_sector_posts.post_message_cid IS NULL;
	COMMENT ON COLUMN miner_sector_posts.sector_id IS NULL;

	COMMENT ON TABLE multisig_transactions IS NULL;
	COMMENT ON COLUMN multisig_transactions.approved IS NULL;
	COMMENT ON COLUMN multisig_transactions.height IS NULL;
	COMMENT ON COLUMN multisig_transactions.method IS NULL;
	COMMENT ON COLUMN multisig_transactions.multisig_id IS NULL;
	COMMENT ON COLUMN multisig_transactions.params IS NULL;
	COMMENT ON COLUMN multisig_transactions.state_root IS NULL;
	COMMENT ON COLUMN multisig_transactions.to IS NULL;
	COMMENT ON COLUMN multisig_transactions.transaction_id IS NULL;
	COMMENT ON COLUMN multisig_transactions.value IS NULL;
`)

	migrations.MustRegisterTx(up, down)
}
