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
	COMMENT ON COLUMN actors.balance IS 'Actor balance in atto-FIL.';
	COMMENT ON COLUMN actors.code IS 'Human readable identifier for the type of the actor.';
	COMMENT ON COLUMN actors.head IS 'CID of the root of the state tree for the actor.';
	COMMENT ON COLUMN actors.height IS 'Epoch when this actor was created or updated.';
	COMMENT ON COLUMN actors.id IS 'Actor address.';
	COMMENT ON COLUMN actors.nonce IS 'The next actor nonce that is expected to appear on chain.';
	COMMENT ON COLUMN actors.state_root IS 'CID of the state root at this epoch.';

	COMMENT ON TABLE block_headers IS 'Blocks included in tipsets at an epoch.';
	COMMENT ON COLUMN block_headers.cid IS 'CID of the block.';
	COMMENT ON COLUMN block_headers.fork_signaling IS 'Flag used as part of signaling forks.';
	COMMENT ON COLUMN block_headers.height IS 'Epoch when this block was mined.';
	COMMENT ON COLUMN block_headers.miner IS 'Address of the miner who mined this block.';
	COMMENT ON COLUMN block_headers.parent_base_fee IS 'The base fee after executing the parent tipset.';
	COMMENT ON COLUMN block_headers.parent_state_root IS 'CID of the block''s parent state root at this epoch.';
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
	COMMENT ON COLUMN chain_economics.burnt_fil IS 'Total FIL (atto-FIL) burned as part of penalties and on-chain computations.';
	COMMENT ON COLUMN chain_economics.circulating_fil IS 'The amount of FIL (atto-FIL) circulating and tradeable in the economy. The basis for Market Cap calculations.';
	COMMENT ON COLUMN chain_economics.locked_fil IS 'The amount of FIL (atto-FIL) locked as part of mining, deals, and other mechanisms.';
	COMMENT ON COLUMN chain_economics.mined_fil IS 'The amount of FIL (atto-FIL) that has been mined by storage miners.';
	COMMENT ON COLUMN chain_economics.parent_state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN chain_economics.vested_fil IS 'Total amount of FIL (atto-FIL) that is vested from genesis allocation.';

	COMMENT ON TABLE chain_powers IS 'Power summaries.';
	COMMENT ON COLUMN chain_powers.height IS 'Epoch this power summary applies to.';
	COMMENT ON COLUMN chain_powers.miner_count IS 'Total number of miners.';
	COMMENT ON COLUMN chain_powers.participating_miner_count IS 'Total number of miners with power above the minimum miner threshold.';
	COMMENT ON COLUMN chain_powers.qa_smoothed_position_estimate IS 'Total power smoothed position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format.';
	COMMENT ON COLUMN chain_powers.qa_smoothed_velocity_estimate IS 'Total power smoothed velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format.';
	COMMENT ON COLUMN chain_powers.state_root IS 'CID of the parent state root at this epoch.';
	COMMENT ON COLUMN chain_powers.total_pledge_collateral IS 'Total locked FIL (atto-FIL) miners have pledged as collateral in order to participate in the economy.';
	COMMENT ON COLUMN chain_powers.total_qa_bytes_committed IS 'Total provably committed, quality adjusted storage power in bytes.';
	COMMENT ON COLUMN chain_powers.total_qa_bytes_power IS 'Total quality adjusted storage power in bytes in the network.';
	COMMENT ON COLUMN chain_powers.total_raw_bytes_committed IS 'Total provably committed storage power in bytes.';
	COMMENT ON COLUMN chain_powers.total_raw_bytes_power IS 'Total storage power in bytes in the network.';
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
`)

	migrations.MustRegisterTx(up, down)
}
