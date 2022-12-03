# actor_states
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| head | text | NO | CID of the root of the state tree for the actor. |
| code | text | NO | CID identifier for the type of the actor. |
| state | jsonb | NO | Top level of state data. |
| height | bigint | NO | Epoch when this state change happened. |
_Actor states that were changed at an epoch. Associates actors states as single-level trees with CIDs pointing to complete state tree with the root CID (head) for that actor's state._
# actors
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| id | text | NO | Actor address. |
| code | text | NO | Human readable identifier for the type of the actor. |
| head | text | NO | CID of the root of the state tree for the actor. |
| nonce | bigint | NO | The next actor nonce that is expected to appear on chain. |
| balance | text | NO | Actor balance in attoFIL. |
| state_root | text | NO | CID of the state root. |
| height | bigint | NO | Epoch when this actor was created or updated. |
_Actors on chain that were added or updated at an epoch. Associates the actor's state root CID (head) with the chain state root CID from which it decends. Includes account ID nonce and balance at each state._
# block_headers
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| cid | text | NO | CID of the block. |
| parent_weight | text | NO | Aggregate chain weight of the block's parent set. |
| parent_state_root | text | NO | CID of the block's parent state root. |
| height | bigint | NO | Epoch when this block was mined. |
| miner | text | NO | Address of the miner who mined this block. |
| timestamp | bigint | NO | Time the block was mined in Unix time, the number of seconds elapsed since January 1, 1970 UTC. |
| win_count | bigint | YES | Number of reward units won in this block. |
| parent_base_fee | text | NO | The base fee after executing the parent tipset. |
| fork_signaling | bigint | NO | Flag used as part of signaling forks. |
_Blocks included in tipsets at an epoch._
# block_messages
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| block | text | NO | CID of the block that contains the message. |
| message | text | NO | CID of a message in the block. |
| height | bigint | NO | Epoch when the block was mined. |
_Message CIDs and the Blocks CID which contain them._
# block_parents
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| block | text | NO | CID of the block. |
| parent | text | NO | CID of the parent block. |
| height | bigint | NO | Epoch when the block was mined. |
_Block CIDs to many parent Block CIDs._
# chain_economics
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch of the economic summary. |
| parent_state_root | text | NO | CID of the parent state root. |
| circulating_fil | numeric | NO | The amount of FIL (attoFIL) circulating and tradeable in the economy. The basis for Market Cap calculations. |
| vested_fil | numeric | NO | Total amount of FIL (attoFIL) that is vested from genesis allocation. |
| mined_fil | numeric | NO | The amount of FIL (attoFIL) that has been mined by storage miners. |
| burnt_fil | numeric | NO | Total FIL (attoFIL) burned as part of penalties and on-chain computations. |
| locked_fil | numeric | NO | The amount of FIL (attoFIL) locked as part of mining, deals, and other mechanisms. |
| fil_reserve_disbursed | numeric | NO | The amount of FIL (attoFIL) that has been disbursed from the mining reserve. |
_Economic summaries per state root CID._
# chain_powers
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| state_root | text | NO | CID of the parent state root. |
| total_raw_bytes_power | numeric | NO | Total storage power in bytes in the network. Raw byte power is the size of a sector in bytes. |
| total_raw_bytes_committed | numeric | NO | Total provably committed storage power in bytes. Raw byte power is the size of a sector in bytes. |
| total_qa_bytes_power | numeric | NO | Total quality adjusted storage power in bytes in the network. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals. |
| total_qa_bytes_committed | numeric | NO | Total provably committed, quality adjusted storage power in bytes. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals. |
| total_pledge_collateral | numeric | NO | Total locked FIL (attoFIL) miners have pledged as collateral in order to participate in the economy. |
| qa_smoothed_position_estimate | numeric | NO | Total power smoothed position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format. |
| qa_smoothed_velocity_estimate | numeric | NO | Total power smoothed velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format. |
| miner_count | bigint | YES | Total number of miners. |
| participating_miner_count | bigint | YES | Total number of miners with power above the minimum miner threshold. |
| height | bigint | NO | Epoch this power summary applies to. |
_Power summaries from the Power actor._
# chain_rewards
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| state_root | text | NO | CID of the parent state root. |
| cum_sum_baseline | numeric | NO | Target that CumsumRealized needs to reach for EffectiveNetworkTime to increase. It is measured in byte-epochs (space * time) representing power committed to the network for some duration. |
| cum_sum_realized | numeric | NO | Cumulative sum of network power capped by BaselinePower(epoch). It is measured in byte-epochs (space * time) representing power committed to the network for some duration. |
| effective_baseline_power | numeric | NO | The baseline power (in bytes) at the EffectiveNetworkTime epoch. |
| new_baseline_power | numeric | NO | The baseline power (in bytes) the network is targeting. |
| new_reward_smoothed_position_estimate | numeric | NO | Smoothed reward position estimate - Alpha Beta Filter "position" (value) estimate in Q.128 format. |
| new_reward_smoothed_velocity_estimate | numeric | NO | Smoothed reward velocity estimate - Alpha Beta Filter "velocity" (rate of change of value) estimate in Q.128 format. |
| total_mined_reward | numeric | NO | The total FIL (attoFIL) awarded to block miners. |
| new_reward | numeric | YES | The reward to be paid in per WinCount to block producers. The actual reward total paid out depends on the number of winners in any round. This value is recomputed every non-null epoch and used in the next non-null epoch. |
| effective_network_time | bigint | YES | Ceiling of real effective network time "theta" based on CumsumBaselinePower(theta) == CumsumRealizedPower. Theta captures the notion of how much the network has progressed in its baseline and in advancing network time. |
| height | bigint | NO | Epoch this rewards summary applies to. |
_Reward summaries from the Reward actor._
# derived_gas_outputs
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| cid | text | NO | CID of the message. |
| from | text | NO | Address of actor that sent the message. |
| to | text | NO | Address of actor that received the message. |
| value | numeric | NO | The FIL value transferred (attoFIL) to the message receiver. |
| gas_fee_cap | numeric | NO | The maximum price that the message sender is willing to pay per unit of gas. |
| gas_premium | numeric | NO | The price per unit of gas (measured in attoFIL/gas) that the message sender is willing to pay (on top of the BaseFee) to "tip" the miner that will include this message in a block. |
| gas_limit | bigint | YES | A hard limit on the amount of gas (i.e., number of units of gas) that a message’s execution should be allowed to consume on chain. It is measured in units of gas. |
| size_bytes | bigint | YES | Size in bytes of the serialized message. |
| nonce | bigint | YES | The message nonce, which protects against duplicate messages and multiple messages with the same values. |
| method | bigint | YES | The method number to invoke. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution. |
| state_root | text | NO | CID of the parent state root. |
| exit_code | bigint | NO | The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific. |
| gas_used | bigint | NO | A measure of the amount of resources (or units of gas) consumed, in order to execute a message. |
| parent_base_fee | numeric | NO | The set price per unit of gas (measured in attoFIL/gas unit) to be burned (sent to an unrecoverable address) for every message execution. |
| base_fee_burn | numeric | NO | The amount of FIL (in attoFIL) to burn as a result of the base fee. It is parent_base_fee (or gas_fee_cap if smaller) multiplied by gas_used. Note: successful window PoSt messages are not charged this burn. |
| over_estimation_burn | numeric | NO | The fee to pay (in attoFIL) for overestimating the gas used to execute a message. The overestimated gas to burn (gas_burned) is a portion of the difference between gas_limit and gas_used. The over_estimation_burn value is gas_burned * parent_base_fee. |
| miner_penalty | numeric | NO | Any penalty fees (in attoFIL) the miner incured while executing the message. |
| miner_tip | numeric | NO | The amount of FIL (in attoFIL) the miner receives for executing the message. Typically it is gas_premium * gas_limit but may be lower if the total fees exceed the gas_fee_cap. |
| refund | numeric | NO | The amount of FIL (in attoFIL) to refund to the message sender after base fee, miner tip and overestimation amounts have been deducted. |
| gas_refund | bigint | NO | The overestimated units of gas to refund. It is a portion of the difference between gas_limit and gas_used. |
| gas_burned | bigint | NO | The overestimated units of gas to burn. It is a portion of the difference between gas_limit and gas_used. |
| height | bigint | NO | Epoch this message was executed at. |
| actor_name | text | NO | Human readable identifier for the type of the actor. |
_Derived gas costs resulting from execution of a message in the VM._
# drand_block_entries
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| round | bigint | NO | The round number of the randomness used. |
| block | text | NO | CID of the block. |
_Drand randomness round numbers used in each block._
# id_addresses
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch at which this address mapping was added. |
| id | text | NO | ID of the actor. |
| address | text | NO | Robust address of the actor. |
| state_root | text | NO | CID of the parent state root at which this address mapping was added. |
_Mapping of IDs to robust addresses from the init actor's state._
# internal_messages
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch this message was executed at. |
| cid | text | NO | CID of the message. |
| state_root | text | NO | CID of the parent state root at which this message was executed. |
| source_message | text | YES | CID of the message that caused this message to be sent. |
| from | text | NO | Address of the actor that sent the message. |
| to | text | NO | Address of the actor that received the message. |
| value | numeric | NO | Amount of FIL (in attoFIL) transferred by this message. |
| method | bigint | NO | The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution. |
| actor_name | text | NO | The full versioned name of the actor that received the message (for example fil/3/storagepower). |
| actor_family | text | NO | The short unversioned name of the actor that received the message (for example storagepower). |
| exit_code | bigint | NO | The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific. |
| gas_used | bigint | NO | A measure of the amount of resources (or units of gas) consumed, in order to execute a message. |
_Messages generated implicitly by system actors and by using the runtime send method._
# internal_parsed_messages
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch this message was executed at. |
| cid | text | NO | CID of the message. |
| from | text | NO | Address of the actor that sent the message. |
| to | text | NO | Address of the actor that received the message. |
| value | numeric | NO | Amount of FIL (in attoFIL) transferred by this message. |
| method | text | NO | The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution. |
| params | jsonb | YES | Method parameters parsed and serialized as a JSON object. |
_Internal messages parsed to extract useful information._
# market_deal_proposals
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| deal_id | bigint | NO | Identifier for the deal. |
| state_root | text | NO | CID of the parent state root for this deal. |
| piece_cid | text | NO | CID of a sector piece. A Piece is an object that represents a whole or part of a File. |
| padded_piece_size | bigint | NO | The piece size in bytes with padding. |
| unpadded_piece_size | bigint | NO | The piece size in bytes without padding. |
| is_verified | boolean | NO | Deal is with a verified provider. |
| client_id | text | NO | Address of the actor proposing the deal. |
| provider_id | text | NO | Address of the actor providing the services. |
| start_epoch | bigint | NO | The epoch at which this deal with begin. Storage deal must appear in a sealed (proven) sector no later than start_epoch, otherwise it is invalid. |
| end_epoch | bigint | NO | The epoch at which this deal with end. |
| storage_price_per_epoch | text | NO | The amount of FIL (in attoFIL) that will be transferred from the client to the provider every epoch this deal is active for. |
| provider_collateral | text | NO | The amount of FIL (in attoFIL) the provider has pledged as collateral. The Provider deal collateral is only slashed when a sector is terminated before the deal expires. |
| client_collateral | text | NO | The amount of FIL (in attoFIL) the client has pledged as collateral. |
| label | text | YES | An arbitrary client chosen label to apply to the deal. |
| height | bigint | NO | Epoch at which this deal proposal was added or changed. |
| is_string | boolean | YES | When true Label contains a valid UTF-8 string encoded in base64. When false Label contains raw bytes encoded in base64. Required by FIP: 27 |
_All storage deal states with latest values applied to end_epoch when updates are detected on-chain._
# market_deal_states
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| deal_id | bigint | NO | Identifier for the deal. |
| sector_start_epoch | bigint | NO | Epoch this deal was included in a proven sector. -1 if not yet included in proven sector. |
| last_update_epoch | bigint | NO | Epoch this deal was last updated at. -1 if deal state never updated. |
| slash_epoch | bigint | NO | Epoch this deal was slashed at. -1 if deal was never slashed. |
| state_root | text | NO | CID of the parent state root for this deal. |
| height | bigint | NO | Epoch at which this deal was added or changed. |
_All storage deal state transitions detected on-chain._
# message_gas_economy
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| state_root | text | NO | CID of the parent state root at this epoch. |
| gas_limit_total | numeric | NO | The sum of all the gas limits. |
| gas_limit_unique_total | numeric | YES | The sum of all the gas limits of unique messages. |
| base_fee | numeric | NO | The set price per unit of gas (measured in attoFIL/gas unit) to be burned (sent to an unrecoverable address) for every message execution. |
| base_fee_change_log | double precision | NO | The logarithm of the change between new and old base fee. |
| gas_fill_ratio | double precision | YES | The gas_limit_total / target gas limit total for all blocks. |
| gas_capacity_ratio | double precision | YES | The gas_limit_unique_total / target gas limit total for all blocks. |
| gas_waste_ratio | double precision | YES | (gas_limit_total - gas_limit_unique_total) / target gas limit total for all blocks. |
| height | bigint | NO | Epoch these economics apply to. |
_Gas economics for all messages in all blocks at each epoch._
# messages
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| cid | text | NO | CID of the message. |
| from | text | NO | Address of the actor that sent the message. |
| to | text | NO | Address of the actor that received the message. |
| size_bytes | bigint | NO | Size of the serialized message in bytes. |
| nonce | bigint | NO | The message nonce, which protects against duplicate messages and multiple messages with the same values. |
| value | numeric | NO | Amount of FIL (in attoFIL) transferred by this message. |
| gas_fee_cap | numeric | NO | The maximum price that the message sender is willing to pay per unit of gas. |
| gas_premium | numeric | NO | The price per unit of gas (measured in attoFIL/gas) that the message sender is willing to pay (on top of the BaseFee) to "tip" the miner that will include this message in a block. |
| method | bigint | YES | The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution. |
| height | bigint | NO | Epoch this message was executed at. |
_Validated on-chain messages by their CID and their metadata._
# miner_current_deadline_infos
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch at which this info was calculated. |
| miner_id | text | NO | Address of the miner this info relates to. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| deadline_index | bigint | NO | A deadline index, in [0..d.WPoStProvingPeriodDeadlines) unless period elapsed. |
| period_start | bigint | NO | First epoch of the proving period (<= CurrentEpoch). |
| open | bigint | NO | First epoch from which a proof may be submitted (>= CurrentEpoch). |
| close | bigint | NO | First epoch from which a proof may no longer be submitted (>= Open). |
| challenge | bigint | NO | Epoch at which to sample the chain for challenge (< Open). |
| fault_cutoff | bigint | NO | First epoch at which a fault declaration is rejected (< Open). |
_Deadline refers to the window during which proofs may be submitted._
# miner_fee_debts
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch at which this debt applies. |
| miner_id | text | NO | Address of the miner that owes fees. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| fee_debt | numeric | NO | Absolute value of debt this miner owes from unpaid fees in attoFIL. |
_Miner debts per epoch from unpaid fees._
# miner_infos
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch at which this miner info was added/changed. |
| miner_id | text | NO | Address of miner this info applies to. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| owner_id | text | NO | Address of actor designated as the owner. The owner address is the address that created the miner, paid the collateral, and has block rewards paid out to it. |
| worker_id | text | NO | Address of actor designated as the worker. The worker is responsible for doing all of the work, submitting proofs, committing new sectors, and all other day to day activities. |
| new_worker | text | YES | Address of a new worker address that will become effective at worker_change_epoch. |
| worker_change_epoch | bigint | NO | Epoch at which a new_worker address will become effective. |
| consensus_faulted_elapsed | bigint | NO | The next epoch this miner is eligible for certain permissioned actor methods and winning block elections as a result of being reported for a consensus fault. |
| peer_id | text | YES | Current libp2p Peer ID of the miner. |
| control_addresses | jsonb | YES | JSON array of control addresses. Control addresses are used to submit WindowPoSts proofs to the chain. WindowPoSt is the mechanism through which storage is verified in Filecoin and is required by miners to submit proofs for all sectors every 24 hours. Those proofs are submitted as messages to the blockchain and therefore need to pay the respective fees. |
| multi_addresses | jsonb | YES | JSON array of multiaddrs at which this miner can be reached. |
_Miner Account IDs for all associated addresses plus peer ID. See https://docs.filecoin.io/mine/lotus/miner-addresses/ for more information._
# miner_locked_funds
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch at which these details were added/changed. |
| miner_id | text | NO | Address of the miner these details apply to. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| locked_funds | numeric | NO | Amount of FIL (in attoFIL) locked due to vesting. When a Miner receives tokens from block rewards, the tokens are locked and added to the Miner's vesting table to be unlocked linearly over some future epochs. |
| initial_pledge | numeric | NO | Amount of FIL (in attoFIL) locked due to it being pledged as collateral. When a Miner ProveCommits a Sector, they must supply an "initial pledge" for the Sector, which acts as collateral. If the Sector is terminated, this deposit is removed and burned along with rewards earned by this sector up to a limit. |
| pre_commit_deposits | numeric | NO | Amount of FIL (in attoFIL) locked due to it being used as a PreCommit deposit. When a Miner PreCommits a Sector, they must supply a "precommit deposit" for the Sector, which acts as collateral. If the Sector is not ProveCommitted on time, this deposit is removed and burned. |
_Details of Miner funds locked and unavailable for use._
# miner_pre_commit_infos
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| miner_id | text | NO | Address of the miner who owns the sector. |
| sector_id | bigint | NO | Numeric identifier for the sector. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| sealed_cid | text | NO | CID of the sealed sector. |
| seal_rand_epoch | bigint | YES | Seal challenge epoch. Epoch at which randomness should be drawn to tie Proof-of-Replication to a chain. |
| expiration_epoch | bigint | YES | Epoch this sector expires. |
| pre_commit_deposit | numeric | NO | Amount of FIL (in attoFIL) used as a PreCommit deposit. If the Sector is not ProveCommitted on time, this deposit is removed and burned. |
| pre_commit_epoch | bigint | YES | Epoch this PreCommit was created. |
| deal_weight | numeric | NO | Total space*time of submitted deals. |
| verified_deal_weight | numeric | NO | Total space*time of submitted verified deals. |
| is_replace_capacity | boolean | YES | Whether to replace a "committed capacity" no-deal sector (requires non-empty DealIDs). |
| replace_sector_deadline | bigint | YES | The deadline location of the sector to replace. |
| replace_sector_partition | bigint | YES | The partition location of the sector to replace. |
| replace_sector_number | bigint | YES | ID of the committed capacity sector to replace. |
| height | bigint | NO | Epoch this PreCommit information was added/changed. |
_Information on sector PreCommits._
# miner_sector_deals
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| miner_id | text | NO | Address of the miner the deal is with. |
| sector_id | bigint | NO | Numeric identifier of the sector the deal is for. |
| deal_id | bigint | NO | Numeric identifier for the deal. |
| height | bigint | NO | Epoch at which this deal was added/updated. |
_Mapping of Deal IDs to their respective Miner and Sector IDs._
# miner_sector_events
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| miner_id | text | NO | Address of the miner who owns the sector. |
| sector_id | bigint | NO | Numeric identifier of the sector. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| event | USER-DEFINED | NO | Name of the event that occurred. |
| height | bigint | NO | Epoch at which this event occurred. |
_Sector events on-chain per Miner/Sector._
# miner_sector_infos
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| miner_id | text | NO | Address of the miner who owns the sector. |
| sector_id | bigint | NO | Numeric identifier of the sector. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| sealed_cid | text | NO | The root CID of the Sealed Sector’s merkle tree. Also called CommR, or "replica commitment". |
| activation_epoch | bigint | YES | Epoch during which the sector proof was accepted. |
| expiration_epoch | bigint | YES | Epoch during which the sector expires. |
| deal_weight | numeric | NO | Integral of active deals over sector lifetime. |
| verified_deal_weight | numeric | NO | Integral of active verified deals over sector lifetime. |
| initial_pledge | numeric | NO | Pledge collected to commit this sector (in attoFIL). |
| expected_day_reward | numeric | NO | Expected one day projection of reward for sector computed at activation time (in attoFIL). |
| expected_storage_pledge | numeric | NO | Expected twenty day projection of reward for sector computed at activation time (in attoFIL). |
| height | bigint | NO | Epoch at which this sector info was added/updated. |
_Latest state of sectors by Miner._
# miner_sector_posts
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| miner_id | text | NO | Address of the miner who owns the sector. |
| sector_id | bigint | NO | Numeric identifier of the sector. |
| height | bigint | NO | Epoch at which this PoSt message was executed. |
| post_message_cid | text | YES | CID of the PoSt message. |
_Proof of Spacetime for sectors._
# multisig_transactions
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch at which this transaction was executed. |
| multisig_id | text | NO | Address of the multisig actor involved in the transaction. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| transaction_id | bigint | NO | Number identifier for the transaction - unique per multisig. |
| to | text | NO | Address of the recipient who will be sent a message if the proposal is approved. |
| value | text | NO | Amount of FIL (in attoFIL) that will be transferred if the proposal is approved. |
| method | bigint | NO | The method number to invoke on the recipient if the proposal is approved. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method exectution. |
| params | bytea | YES | CBOR encoded bytes of parameters to send to the method that will be invoked if the proposal is approved. |
| approved | jsonb | NO | Addresses of signers who have approved the transaction. 0th entry is the proposer. |
_Details of pending transactions involving multisig actors._
# parsed_messages
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| cid | text | NO | CID of the message. |
| height | bigint | NO | Epoch this message was executed at. |
| from | text | NO | Address of the actor that sent the message. |
| to | text | NO | Address of the actor that received the message. |
| value | numeric | NO | Amount of FIL (in attoFIL) transferred by this message. |
| method | text | NO | The name of the method that was invoked on the recipient actor. |
| params | jsonb | YES | Method parameters parsed and serialized as a JSON object. |
_Messages parsed to extract useful information._
# power_actor_claims
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch this claim was made. |
| miner_id | text | NO | Address of miner making the claim. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| raw_byte_power | numeric | NO | Sum of raw byte storage power for a miner's sectors. Raw byte power is the size of a sector in bytes. |
| quality_adj_power | numeric | NO | Sum of quality adjusted storage power for a miner's sectors. Quality adjusted power is a weighted average of the quality of its space and it is based on the size, duration and quality of its deals. |
_Miner power claims recorded by the power actor._
# receipts
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| message | text | NO | CID of the message this receipt belongs to. |
| state_root | text | NO | CID of the parent state root that this epoch. |
| idx | bigint | NO | Index of message indicating execution order. |
| exit_code | bigint | NO | The exit code that was returned as a result of executing the message. Exit code 0 indicates success. Codes 0-15 are reserved for use by the runtime. Codes 16-31 are common codes shared by different actors. Codes 32+ are actor specific. |
| gas_used | bigint | NO | A measure of the amount of resources (or units of gas) consumed, in order to execute a message. |
| height | bigint | NO | Epoch the message was executed and receipt generated. |
_Message reciepts after being applied to chain state by message CID and parent state root CID of tipset when message was executed._
# verified_registry_verifiers
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch at which this verifiers state changed. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| address | text | NO | Address of verifier this state change applies to. |
| data_cap | numeric | NO | DataCap of verifier at this state change. |
| event | USER-DEFINED | NO | Name of the event that occurred. |
_Verifier on-chain per each verifier state change._
# verified_registry_verified_clients
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| height | bigint | NO | Epoch at which this verified client state changed. |
| state_root | text | NO | CID of the parent state root at this epoch. |
| address | text | NO | Address of verified client this state change applies to. |
| data_cap | numeric | NO | DataCap of verified client at this state change. |
| event | USER-DEFINED | NO | Name of the event that occurred. |
_Verifier on-chain per each verified client state change._
# surveyed_peer_agents
| Column Name | Data Type | Is Nullable | Description |
| --- | --- | --- | --- |
| surveyer_peer_id | text | NO | Peer ID of the node performing the survey. |
| observed_at | timestamp with time zone | NO | Timestamp of the observation. |
| raw_agent | text | NO | Unprocessed agent string as reported by a peer. |
| normalized_agent | text | NO | Agent string normalized to a software name with major and minor version. |
| count | bigint | NO | Number of peers that reported the same raw agent. |
_Observations of filecoin peer agent strings over time._
# vm_messages
| Column Name | Data Type | Is Nullable | Description                                                                                                                                                                     |
|-------------|-----------|-------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| height      | bigint    | NO          | Height message was executed at.                                                                                                                                                 |
| state_root  | text      | NO          | CID of the parent state root at which this message was executed.                                                                                                                |
| cid         | text      | NO          | CID of the message (note this CID does not appear on chain).                                                                                                                    |
| source      | text      | NO          | CID of the on-chain message or implicit (internal) message that caused this message to be sent.                                                                                 |
| from        | text      | NO          | Address of the actor that sent the message.                                                                                                                                     |
| to          | text      | NO          | Address of the actor that received the message.                                                                                                                                 |
| value       | numeric   | NO          | Amount of FIL (in attoFIL) transferred by this message.                                                                                                                         |
| method      | bigint    | NO          | The method number invoked on the recipient actor. Only unique to the actor the method is being invoked on. A method number of 0 is a plain token transfer - no method execution |
| actor_code  | text      | NO          | The CID of the actor that received the message.                                                                                                                                 |
| exit_code   | bigint    | NO          | The exit code that was returned as a result of executing the message.                                                                                                           |
| gas_used    | bigint    | NO          | A measure of the amount of resources (or units of gas) consumed, in order to execute a message.                                                                                 |
| params      | jsonb     | YES         | Message parameters parsed and serialized as a JSON object.                                                                                                                      |
| returns     | jsonb     | YES         | Result returned from executing a message parsed and serialized as a JSON object.                                                                                                |
# surveyed_miner_protocols
| Column Name | Data Type                | Is Nullable | Description                                                     |
|-------------|--------------------------|-------------|-----------------------------------------------------------------|
| observed_at | timestamp with time zone | NO          | Timestamp of the observation.                                   |
| miner_id    | text                     | NO          | Address (ActorID) of the miner.                                 |
| peer_id     | text                     | YES         | PeerID of the miner advertised in on-chain MinerInfo structure. |
| agent       | text                     | YES         | Agent string as reported by the peer.                           |
| protocols   | jsonb                    | YES         | List of supported protocol strings supported by the peer.       |
_Observations of Filecoin storage provider supported protocols and agents over time._
# data_cap_balances
| Column Name | Data Type    | Is Nullable | Description                                                      |
|-------------|--------------|-------------|------------------------------------------------------------------|
| height      | bigint       | NO          | Epoch at which DataCap balances state changed.                   |
| state_root  | text         | NO          | CID of the parent state root at this epoch.                      |
| address     | text         | NO          | Address of verified datacap client this state change applies to. |
| data_cap    | numeric      | NO          | DataCap of verified datacap client at this state change.         |
| event       | USER-DEFINED | NO          | Name of the event that occurred (ADDED, MODIFIED, REMOVED).      |
_DataCap balances on-chain per each DataCap state change._
# miner_sector_infos_v7
| Column Name             | Data Type | Is Nullable | Description                                                                                   |
|-------------------------|-----------|-------------|-----------------------------------------------------------------------------------------------|
| miner_id                | text      | NO          | Address of the miner who owns the sector.                                                     |
| sector_id               | bigint    | NO          | Numeric identifier of the sector.                                                             |
| state_root              | text      | NO          | CID of the parent state root at this epoch.                                                   |
| sealed_cid              | text      | NO          | The root CID of the Sealed Sector’s merkle tree. Also called CommR, or "replica commitment".  |
| activation_epoch        | bigint    | YES         | Epoch during which the sector proof was accepted.                                             |
| expiration_epoch        | bigint    | YES         | Epoch during which the sector expires.                                                        |
| deal_weight             | numeric   | NO          | Integral of active deals over sector lifetime.                                                |
| verified_deal_weight    | numeric   | NO          | Integral of active verified deals over sector lifetime.                                       |
| initial_pledge          | numeric   | NO          | Pledge collected to commit this sector (in attoFIL).                                          |
| expected_day_reward     | numeric   | NO          | Expected one day projection of reward for sector computed at activation time (in attoFIL).    |
| expected_storage_pledge | numeric   | NO          | Expected twenty day projection of reward for sector computed at activation time (in attoFIL). |
| height                  | bigint    | NO          | Epoch at which this sector info was added/updated.                                            |
| sector_key_cid          | text      | YES         | SealedSectorCID is set when CC sector is snapped.                                             |
_Latest state of sectors by Miner for actors v7 and above._
