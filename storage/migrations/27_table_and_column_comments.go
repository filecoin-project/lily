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
	COMMENT ON COLUMN drand_block_entries.round IS 'The round number of the randomness used.';
	COMMENT ON COLUMN drand_block_entries.block IS 'CID of the block.';
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
	COMMENT ON COLUMN drand_block_entries.round IS NULL;
	COMMENT ON COLUMN drand_block_entries.block IS NULL;
`)

	migrations.MustRegisterTx(up, down)
}
