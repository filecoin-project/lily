# Changelog
All notable changes to this project will be documented in this file.

The format is a variant of [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) combined with categories from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Breaking changes should trigger an increment to the major version. Features increment the minor version and fixes or other changes increment the patch number.

<a name="v0.24.1"></a>



## [v0.24.1] - 2025-09-25

### Fix
* fix: support miner_info in v17 (https://github.com/filecoin-project/lily/pull/1365)

## [v0.24.0] - 2025-09-23

### Feat
* feat: support cron burn (https://github.com/filecoin-project/lily/pull/1361)

### Chore
* chore: update lotus to v1.34.0 (https://github.com/filecoin-project/lily/pull/1362)
* chore: update lotus version to v1.34.1 (https://github.com/filecoin-project/lily/pull/1363)


## [v0.23.4] - 2025-06-03

### Chore
* chore: update lotus to v1.33.0 (https://github.com/filecoin-project/lily/pull/1356)

### Fix
* fix: replace eth.EthTransactionAPI with full.EthTransactionAPIV1 (https://github.com/filecoin-project/lily/pull/1358)

## [v0.23.3] - 2025-05-14

### Fix
* fix: the wrong logging and return in vm_messages for parsing return and params (https://github.com/filecoin-project/lily/pull/1352)


### Chore
* chore: update lotus (https://github.com/filecoin-project/lily/pull/1350)
* chore: clean the log in miner_actor_dump (https://github.com/filecoin-project/lily/pull/1351)
* chore: clean the useless logging in sectorv7 (https://github.com/filecoin-project/lily/pull/1353)


## DATABASE MIGRATION
* [1.45](https://github.com/filecoin-project/lily/blob/master/schemas/v1/45_add_fields_in_miner_dump.go)

### Feat
* feat: update termination fee (https://github.com/filecoin-project/lily/pull/1344)
* feat: add new column for miner actor dump - termination_fee_v2 (https://github.com/filecoin-project/lily/pull/1345)

### Fix
* fix: migration scripts error (https://github.com/filecoin-project/lily/pull/1346)
* fix: the formula error (https://github.com/filecoin-project/lily/pull/1347)

## [v0.23.1] - 2025-04-15

### Fix:
* fix: nil pointer (https://github.com/filecoin-project/lily/pull/1342)

## [v0.23.0] - 2025-04-14

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.41](https://github.com/filecoin-project/lily/blob/master/schemas/v1/41_add_fields_in_miner_sectro_info.go)
* [1.42](https://github.com/filecoin-project/lily/blob/master/schemas/v1/42_add_fields_in_miner_dump.go)
* [1.43](https://github.com/filecoin-project/lily/blob/master/schemas/v1/43_add_daily_fee_in_miner_sectro_info.go)
* [1.44](https://github.com/filecoin-project/lily/blob/master/schemas/v1/44_add_daily_fee_in_miner_dump.go)

### Feat
* feat: add new fields in miner_sector_info_v7 (https://github.com/filecoin-project/lily/pull/1333)
* feat: add new fields in miner actor dumps (https://github.com/filecoin-project/lily/pull/1334)
* feat: refine the insert batch size (https://github.com/filecoin-project/lily/pull/1335)

### Chore
* chore: update lotus to v1.32.0-rc3 (https://github.com/filecoin-project/lily/pull/1338)
* chore: update lotus to v1.32.2 (https://github.com/filecoin-project/lily/pull/1339)

## [v0.22.0] - 2024-12-13

### Feat
* feat: update the new actor version for specific tasks (https://github.com/filecoin-project/lily/pull/1325)
* feat: update market actor gen (https://github.com/filecoin-project/lily/pull/1326)

### Fix
* fix: insert actor methods db error (https://github.com/filecoin-project/lily/pull/1331)

### Chore
* chore: upgrade the lotus to v1.32.0-rc1 for nv25 (https://github.com/filecoin-project/lily/pull/1329)


## [v0.21.1] - 2024-11-19

### Feat
* feat: update the new actor version for specific tasks by @Terryhung in https://github.com/filecoin-project/lily/pull/1325
* feat: update market actor gen by @Terryhung in https://github.com/filecoin-project/lily/pull/1326


## [v0.21.0] - 2024-11-13

### Feat
* feat: update README.md by @Terryhung in https://github.com/filecoin-project/lily/pull/1314
* feat: upgrade filecoin-ffi version by @Terryhung in https://github.com/filecoin-project/lily/pull/1322


### Chore
* chore: upgrade lotus to v1.29.1 by @Terryhung in https://github.com/filecoin-project/lily/pull/1318
* chore: upgrade lotus version and golang version by @Terryhung in https://github.com/filecoin-project/lily/pull/1320
* chore: upgrade lotus to v1.30.0 for nv24 by @Terryhung in https://github.com/filecoin-project/lily/pull/1321

### Fix
* fix: the error of logic and return by @Terryhung in https://github.com/filecoin-project/lily/pull/1313
* fix: downgrade the bls version to stable version by @Terryhung in https://github.com/filecoin-project/lily/pull/1323

## [v0.20.2] - 2024-08-01

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.40](https://github.com/filecoin-project/lily/blob/master/schemas/v1/40_add_chaineconomics_v2.go)

### feat
* feat: add new table for nv23 by @Terryhung in https://github.com/filecoin-project/lily/pull/1311


## [v0.20.1] - 2024-07-30

### Chore
* chore: upgrade lotus to v1.28.1 (https://github.com/filecoin-project/lily/pull/1309)

<a name="v0.20.0"></a>

## [v0.20.0] - 2024-07-26

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.39](https://github.com/filecoin-project/lily/blob/master/schemas/v1/39_add_locked_fil_v2_chaineconomics.go)

### Feat
* feat: add new column for locked_fil_v2 in chain_economics and address_type in data_cap_balances (https://github.com/filecoin-project/lily/pull/1306)

### Chore
* chore: upgrade the lotus version to 1.27.0 (https://github.com/filecoin-project/lily/pull/1303)
* chore: upgrade lotus to 1.27.1 (https://github.com/filecoin-project/lily/pull/1304)
* chore: update the actor versions (https://github.com/filecoin-project/lily/pull/1305)
* chore: update lotus version for nv23 (1.28.0-rc5) (https://github.com/filecoin-project/lily/pull/1307)

<a name="v0.19.0"></a>

## [v0.19.0] - 2024-05-31

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.37](https://github.com/filecoin-project/lily/blob/master/schemas/v1/37_change_pkey_builtin_actor_event.go)
* [1.38](https://github.com/filecoin-project/lily/blob/master/schemas/v1/38_miner_sector_info_v2.go)

### Feat
* feat: add new column for builtin_actor_events (https://github.com/filecoin-project/lily/pull/1290)
* feat: add new task miner sector deal v2 (https://github.com/filecoin-project/lily/pull/1293)
* feat: refine the provider sectors func (https://github.com/filecoin-project/lily/pull/1294)
* feat: Refine the filter for the GetActorEventsRaw (https://github.com/filecoin-project/lily/pull/1295)
* feat: refine the miner sector event by using builtin_actor_event function (https://github.com/filecoin-project/lily/pull/1298)
* feat: refine miner_sector_deal_v2 performance issue (https://github.com/filecoin-project/lily/pull/1300)
* feat: refactor builtin actor event (https://github.com/filecoin-project/lily/pull/1301)

### Fix
* fix: use the new library to parse cid from builtin_actor_event (https://github.com/filecoin-project/lily/pull/1292)


<a name="v0.18.0"></a>

## [v0.18.0] - 2024-04-23

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.36](https://github.com/filecoin-project/lily/blob/master/schemas/v1/36_builtin_actor_event.go)

### Feat
* feat: Refine the market diff function (https://github.com/filecoin-project/lily/pull/1286)
* feat: add builtin-actor-event task (https://github.com/filecoin-project/lily/pull/1287)

### Chore
* chore: upgrade the lotus version to v1.26.3 (https://github.com/filecoin-project/lily/pull/1288)


<a name="v0.17.4"></a>

## [v0.17.5 - 2024-03-28]

### Chore
* chore: update the lotus to 1.26.1

  

<a name="v0.17.3"></a>

## [v0.17.3] - 2023-12-11

### Fix
* fix: use exponential backoff to attempt reconnect to tsdb (https://github.com/filecoin-project/lily/pull/1267)

### Chore
* chore: update the lotus to 1.25.0 (https://github.com/filecoin-project/lily/pull/1274)


<a name="v0.17.2"></a>
## [v0.17.2] - 2023-10-23

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.33](https://github.com/filecoin-project/lily/blob/master/schemas/v1/33_miner_actor_dumps.go)
* [1.34](https://github.com/filecoin-project/lily/blob/master/schemas/v1/34_unsynced_block_headers.go)
* [1.35](https://github.com/filecoin-project/lily/blob/master/schemas/v1/35_add_pkey_for_tables.go)
  
### Feat
* feat: add new actor dump: miner (https://github.com/filecoin-project/lily/pull/1264)
  * requires migration [1.33](https://github.com/filecoin-project/lily/blob/master/schemas/v1/33_miner_actor_dumps.go)
* feat: add new job to find out orphan block (https://github.com/filecoin-project/lily/pull/1265)
  * requires migration [1.34](https://github.com/filecoin-project/lily/blob/master/schemas/v1/34_unsynced_block_headers.go)
* feat: Add new function for filtering invalid message (https://github.com/filecoin-project/lily/pull/1269)
* feat: add the cache for robust address by @Terryhung in https://github.com/filecoin-project/lily/pull/1273

### Fix
* fix: remove the typo in fevm transactions model (https://github.com/filecoin-project/lily/pull/1262)
* fix: the peer_id encode error (https://github.com/filecoin-project/lily/pull/1268)
* fix: align the pkey in model to real database schema (https://github.com/filecoin-project/lily/pull/1271)
  * requires migration [1.35](https://github.com/filecoin-project/lily/blob/master/schemas/v1/35_add_pkey_for_tables.go)
* fix: hot fix migration script (https://github.com/filecoin-project/lily/pull/1272)

### Chore
* chore: update the lotus version to 1.23.3 (https://github.com/filecoin-project/lily/pull/1251)
* chore(deps): bump actions/checkout from 3 to 4 (https://github.com/filecoin-project/lily/pull/1263)


<a name="v0.17.1"></a>
## [v0.17.1] - 2023-8-30

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.30](https://github.com/filecoin-project/lily/blob/master/schemas/v1/30_fevm_actor_dumps.go)
* [1.31](https://github.com/filecoin-project/lily/blob/master/schemas/v1/31_add_message_and_address_and_actor_names_to_fevm_transcation.go)
* [1.32](https://github.com/filecoin-project/lily/blob/master/schemas/v1/32_change_the_fevm_trace_pkey.go)

  
### Feat
* feat: add actor dump processor and task (https://github.com/filecoin-project/lily/pull/1245)
  * requires migration [1.30](https://github.com/filecoin-project/lily/blob/master/schemas/v1/30_fevm_actor_dumps.go)
* feat: add new config to prefect service (https://github.com/filecoin-project/lily/pull/1257)
* feat: add more fields in fevm transaction (https://github.com/filecoin-project/lily/pull/1255)
  * requires migration [1.31](https://github.com/filecoin-project/lily/blob/master/schemas/v1/31_add_message_and_address_and_actor_names_to_fevm_transcation.go)

  
### Fix
* fix: align the eth address in fevm trace (https://github.com/filecoin-project/lily/pull/1253)
* fix: transaction hash in fevm trace (https://github.com/filecoin-project/lily/pull/1254)
* fix: skip StateListActors if there is no actor dump processor (https://github.com/filecoin-project/lily/pull/1256)

### Chore
* chore: Change the pkey of fevm_traces and refine the previous migration script (https://github.com/filecoin-project/lily/pull/1259)
  * requires migration [1.32](https://github.com/filecoin-project/lily/blob/master/schemas/v1/32_change_the_fevm_trace_pkey.go)


<a name="v0.17.0"></a>
## [v0.17.0] - 2023-8-1

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.22](https://github.com/filecoin-project/lily/blob/master/schemas/v1/22_fevm_block_header.go)
* [1.23](https://github.com/filecoin-project/lily/blob/master/schemas/v1/23_fevm_receipt.go)
* [1.24](https://github.com/filecoin-project/lily/blob/master/schemas/v1/24_fevm_transaction.go)
* [1.25](https://github.com/filecoin-project/lily/blob/master/schemas/v1/25_fevm_contract.go)
* [1.26](https://github.com/filecoin-project/lily/blob/master/schemas/v1/26_fevm_rename_table.go)
* [1.27](https://github.com/filecoin-project/lily/blob/master/schemas/v1/27_fevm_traces.go)
* [1.28](https://github.com/filecoin-project/lily/blob/master/schemas/v1/28_add_actor_names_to_fevm_traces.go)
* [1.29](https://github.com/filecoin-project/lily/blob/master/schemas/v1/29_add_parsed_return_to_receipt_returns.go)

### Feat
* feat: add new task: fevm_block_header (https://github.com/filecoin-project/lily/pull/1207)
  * requires migration [1.22](https://github.com/filecoin-project/lily/blob/master/schemas/v1/22_fevm_block_header.go)
* feat: add new task for fevm receipt (https://github.com/filecoin-project/lily/pull/1208)
  * requires migration [1.23](https://github.com/filecoin-project/lily/blob/master/schemas/v1/23_fevm_receipt.go)
* feat: add new option for walk config (https://github.com/filecoin-project/lily/pull/1212)
* feat: add new task: fevm_transaction (https://github.com/filecoin-project/lily/pull/1213)
  * requires migration [1.24](https://github.com/filecoin-project/lily/blob/master/schemas/v1/24_fevm_transaction.go)
* feat: add new task fevm contract (https://github.com/filecoin-project/lily/pull/1214)
  * requires migration [1.25](https://github.com/filecoin-project/lily/blob/master/schemas/v1/25_fevm_contract.go)
* feat: add new task for fevm trace (https://github.com/filecoin-project/lily/pull/1217)
  * requires migration [1.27](https://github.com/filecoin-project/lily/blob/master/schemas/v1/27_fevm_traces.go)
* feat: add more description for fevm models (https://github.com/filecoin-project/lily/pull/1231)
* feat: change the logging condition for parsing internal message (https://github.com/filecoin-project/lily/pull/1243)
* feat: add human readable actor codes in fevm_traces (https://github.com/filecoin-project/lily/pull/1240)
  * requires migration [1.28](https://github.com/filecoin-project/lily/blob/master/schemas/v1/28_add_actor_names_to_fevm_traces.go)
* feat: change the splitstore setting in config and replace returning error with  logging error in parsing vm_message ( https://github.com/filecoin-project/lily/pull/1246)
* feat: add new columns for receipts table (https://github.com/filecoin-project/lily/pull/1248)
  * requires migration [1.29](https://github.com/filecoin-project/lily/blob/master/schemas/v1/29_add_parsed_return_to_receipt_returns.go)
  
### Fix
* fix: CodeCID should be at end of model (https://github.com/filecoin-project/lily/pull/1205)
* fix: batch insert for locked_funds, sector_events and sector_info_v7 (https://github.com/filecoin-project/lily/pull/1216)
* fix: batch insert miner_pre_commit_info (https://github.com/filecoin-project/lily/pull/1220)
* fix: run more backfill tipsetkey during import chain (https://github.com/filecoin-project/lily/pull/1222)
* fix: keep running the whole tipset task without cancel (https://github.com/filecoin-project/lily/pull/1219)
* fix: rename the fevm related table (https://github.com/filecoin-project/lily/pull/1218)
  * requires migration [1.26](https://github.com/filecoin-project/lily/blob/master/schemas/v1/26_fevm_rename_table.go)
* fix: align the task height (https://github.com/filecoin-project/lily/pull/1225)
* fix: logic error and remove useless function call (https://github.com/filecoin-project/lily/pull/1227)
* fix: fix the logic and data type in fevm tasks (https://github.com/filecoin-project/lily/pull/1232)
* fix: refine error handling for skipping actor limit (https://github.com/filecoin-project/lily/pull/1237)
* fix: var names for chain commands (https://github.com/filecoin-project/lily/pull/1236)
* fix: resize error channel (https://github.com/filecoin-project/lily/pull/1238)
* fix: add Transform implementation for PreCommitInfoExtractorV8 (https://github.com/filecoin-project/lily/pull/1242)

### Chore
* chore: batch insert miner_sector_deal (https://github.com/filecoin-project/lily/pull/1221)
* chore: add fevm supported tasks and refine the migration script (https://github.com/filecoin-project/lily/pull/1228)
* chore(deps): bump actions/checkout from 2 to 3 (https://github.com/filecoin-project/lily/pull/1230)
* chore(deps): bump github/codeql-action from 1 to 2 (https://github.com/filecoin-project/lily/pull/1229)
* chore: do not require mac users to preset env vars (https://github.com/filecoin-project/lily/pull/1239)
* chore: update the lotus version to 1.23.2 (https://github.com/filecoin-project/lily/pull/1241)


<a name="v0.16.0"></a>
## [v0.16.0] - 2023-5-15

### Feat
* feat: add single lily node deployment (https://github.com/filecoin-project/lily/pull/1203)
* feat: add new command: prune (https://github.com/filecoin-project/lily/pull/1198)
* feat: add filter for actor changes (https://github.com/filecoin-project/lily/pull/1195)

### Fix
* fix: batch insert actor & actor_state data (https://github.com/filecoin-project/lily/pull/1192)
* fix: add missing span.End() (https://github.com/filecoin-project/lily/pull/1202)

### Chore
* chore: Update the actor codes versions (https://github.com/filecoin-project/lily/pull/1193)
* chore: publish images for every master commit (https://github.com/filecoin-project/lily/pull/1196)
* chore: move persist duration timer to ExportResult (https://github.com/filecoin-project/lily/pull/1197)
* chore: update go-amt-ipld (https://github.com/filecoin-project/lily/pull/1199)
* chore: update go-hamt-ipld (https://github.com/filecoin-project/lily/pull/1200)
  

<a name="v0.15.1"></a>
## [v0.15.1] - 2023-4-25

### Fix
* fix: using the env variable for lily redis (https://github.com/filecoin-project/lily/pull/1185)


### Chore
* chore: update ffi version (https://github.com/filecoin-project/lily/pull/1184)


<a name="v0.15.0"></a>
## [v0.15.0] - 2023-4-25

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.19](https://github.com/filecoin-project/lily/blob/master/schemas/v1/19_fevm_actor_stats.go)
* [1.20](https://github.com/filecoin-project/lily/blob/master/schemas/v1/20_actor_mappings.go)
* [1.21](https://github.com/filecoin-project/lily/blob/master/schemas/v1/21_actor_add_state.go)

### Feat
* feat: Add retry opt for asynq https://github.com/filecoin-project/lily/pull/1167
* Create FEVM actor stats task https://github.com/filecoin-project/lily/pull/1173
  * implement fevm_actor_stats task and schema
    * task name `fevm_actor_stats`
    * requires migration [1.19](https://github.com/filecoin-project/lily/blob/master/schemas/v1/19_fevm_actor_stats.go)
* feat: add cmd to print actor codes and methods in CSV format to stdout https://github.com/filecoin-project/lily/pull/1155
  * implement cmd for storing actor codes and methods
    * cmd `lily chain actor-codes` 
    * requires migration [1.20](https://github.com/filecoin-project/lily/blob/master/schemas/v1/20_actor_mappings.go)
* feat: add more tasks for notifier default tasks https://github.com/filecoin-project/lily/pull/1178
* feat: Add the state and code_cid for actors table https://github.com/filecoin-project/lily/pull/1174
  * add the new field in actors
    * requires migration [1.21](https://github.com/filecoin-project/lily/blob/master/schemas/v1/21_actor_add_state.go)
* feat: add grafana agent and refactor configs https://github.com/filecoin-project/lily/pull/1180
* Read redis address & username from env https://github.com/filecoin-project/lily/pull/1179


### Fix
* fix: log unknown actor type without returning error https://github.com/filecoin-project/lily/pull/1169
* fix: update the go-state-types for resolving vm message task error https://github.com/filecoin-project/lily/pull/1170
* fix: do not error on null or empty params for parsing message https://github.com/filecoin-project/lily/pull/1171

### Chore
* feat:update the lotus version https://github.com/filecoin-project/lily/pull/1153
* feat: update the lotus version to v1.23.0 https://github.com/filecoin-project/lily/pull/1175


<a name="v0.14.1"></a>
## [v0.14.1] - 2023-3-16

### Feat
* Support importing zstd snapshot directly ([#1156](https://github.com/filecoin-project/lily/pull/1156))

### Fix
* fix: Fix the error jsonp format in vm_message ([#1160](https://github.com/filecoin-project/lily/pull/1160))
* fix: Add function for cleaning the Unicode escape sequence ([#1161](https://github.com/filecoin-project/lily/pull/1161))
* fix: process actor version 10 tasks ([#1166](https://github.com/filecoin-project/lily/pull/1166))
* fix: prevent unsuccessful task from retrying ([#1162](https://github.com/filecoin-project/lily/pull/1162))
* fix: don't error on unknown method numbers for FEVM ([#1165](https://github.com/filecoin-project/lily/pull/1165))

### Chore
* Use 28e3cd44d91681c0 release from filecoin-ffi ([#1158](https://github.com/filecoin-project/lily/pull/1158))

<a name="v0.14.0"></a>
## [v0.14.0] - 2023-3-8

## DATABASE MIGRATION
This release includes a migration which must be applied before deploying against an existing database.
* [1.13](https://github.com/filecoin-project/lily/blob/master/schemas/v1/13_vm_messages_index.go)
* [1.14](https://github.com/filecoin-project/lily/blob/master/schemas/v1/14_actor_events.go)
* [1.15](https://github.com/filecoin-project/lily/blob/master/schemas/v1/15_message_params.go)
* [1.16](https://github.com/filecoin-project/lily/blob/master/schemas/v1/16_receipt_returns.go)
* [1.17](https://github.com/filecoin-project/lily/blob/master/schemas/v1/17_verifreg_claims.go)
* [1.18](https://github.com/filecoin-project/lily/blob/master/schemas/v1/18_vm_messages_migration.go)

### Feat
* update command to print all actor codes and version ([#1111](https://github.com/filecoin-project/lily/pull/1111))
* [command] add currentheight to job list ([#1131](https://github.com/filecoin-project/lily/pull/1131))
* add logic for connecting and disconnecting from peers ([1133](https://github.com/filecoin-project/lily/pull/1133))
* metrics for state & block store caches ([#1136](https://github.com/filecoin-project/lily/pull/1136))
* upgrade to lotus v.1.20.0 with support for network version 18 (FEVM) ([#1140](https://github.com/filecoin-project/lily/pull/1140))
  * implement actorevent task and schema
    * task name `actor_events`
    * requires migration [1.14](https://github.com/filecoin-project/lily/blob/master/schemas/v1/14_actor_events.go)
    * parses and persists changes to actor events structure in nv18
  * implement messageparam task and schema
    * task name `message_params`
    * requires migration [1.15](https://github.com/filecoin-project/lily/blob/master/schemas/v1/15_message_params.go)
    * parses and persists changes to actor events structure in nv18
  * implement receiptreturn task and schema
    * task name `receipt_returns`
    * requires migration [1.16](https://github.com/filecoin-project/lily/blob/master/schemas/v1/16_receipt_returns.go)
    * parses and persists changes to actor events structure in nv18
  * implement claim extractor and schema
    * task name `verified_registry_claim`
    * requires migration [1.17](https://github.com/filecoin-project/lily/blob/master/schemas/v1/17_verifreg_claims.go)
    * parses and persists changes to actor events structure in nv18
* Add docker compose file for notifier and worker ([#1138](https://github.com/filecoin-project/lily/pull/1138))

### Fix
* only parse vm messages with exit code OK ([#1112](https://github.com/filecoin-project/lily/pull/1112))
* bug in verifreg verifier event tracking ([#1124](https://github.com/filecoin-project/lily/pull/1124))
* add index column constraint to vm_messages table ([#1127](https://github.com/filecoin-project/lily/pull/1127))
* move the models-* subcmd to a different cmd ([#1135](https://github.com/filecoin-project/lily/pull/1135))
* reduce time bucket size ([#1143](https://github.com/filecoin-project/lily/pull/1143))

### Chore
* Cache proof params in production docker image by @placer14 in https://github.com/filecoin-project/lily/pull/1101
* Migrate vm_messages to hypertable ([#1149](https://github.com/filecoin-project/lily/pull/1149))
  * requires [1.18](https://github.com/filecoin-project/lily/blob/master/schemas/v1/18_vm_messages_migration.go)

<a name="v0.13.0"></a>
## [v0.13.0] - 2022-11-28

## DATABASE MIGRATION

This release includes a migration which must be applied before deploying against an existing database.
* [1.10](https://github.com/filecoin-project/lily/blob/v0.12.0/schemas/v1/10_miner_precommit_v9.go)
* [1.11](https://github.com/filecoin-project/lily/blob/v0.13.0/schemas/v1/11_miner_beneficiary.go)
* [1.12](https://github.com/filecoin-project/lily/blob/v0.13.0/schemas/v1/12_datacap_balance.go)

### Feat
* upgrade to lotus v1.18.0 with support for network version 17 ([#1080](https://github.com/filecoin-project/lily/pull/1080))
  * implement precommitinfov9 task and schema
    * task name `miner_pre_commit_info`
    * requires migration [1.10](https://github.com/filecoin-project/lily/blob/v0.12.0/schemas/v1/10_miner_precommit_v9.go)
    * parses and persists changes to miner PreCommit structure in nv17
  * implement miner beneficiary task and schema
    * task name `miner_beneficiary`
    * requires migration [1.11](https://github.com/filecoin-project/lily/blob/v0.13.0/schemas/v1/11_miner_beneficiary.go)
    * parses and persists miner beneficiary address added to miner info in nv17
  * implement datacap balance task and schema
    * task name `data_cap_balance`
    * requires migration  [1.12](https://github.com/filecoin-project/lily/blob/v0.13.0/schemas/v1/12_datacap_balance.go)
    * parses and persists datacap actor balances from singleton datacap actor in nv17

### Fix
* metrics & debugging link ([#1070](https://github.com/filecoin-project/lily/pull/1070))
* use bitfield.MultiMerge to improve LoadSectorState performance ([#1086](https://github.com/filecoin-project/lily/pull/1086))
* don't parse VM messages whose parent failed to apply ([#1091](https://github.com/filecoin-project/lily/pull/1091))
* remove actor code mapping ([#1087](https://github.com/filecoin-project/lily/pull/1087))

### Chore
* fix metric name ([#1085](https://github.com/filecoin-project/lily/pull/1085))
* create docker images for commits in `cd/staging` ([#1090](https://github.com/filecoin-project/lily/pull/1090))

<a name="v0.12.0"></a>
## [v0.12.0] - 2022-09-07

## DATABASE MIGRATION

This release includes a migration which must be applied before deploying against an existing database.
* [1.8](https://github.com/filecoin-project/lily/blob/v0.12.0/schemas/v1/8_vm_messages.go)
* [1.9](https://github.com/filecoin-project/lily/blob/master/schemas/v1/9_miner_protocols.go)

### Feat
* implement vm message extraction [#1027](https://github.com/filecoin-project/lily/pull/1027)
  * task name `vm_messages`
  * requires migration
  * requires LOTUS_VM_ENABLE_TRACING=1 to be set on daemon executing task
* implement miner protocol survey task [#1048](https://github.com/filecoin-project/lily/pull/1048)
    * task name `minerprotocols`
    * requires migration
    * for use with the survey command.
* add method for finding oldest state and computing state [#1038](https://github.com/filecoin-project/lily/pull/1038)
    * commands:
        * `$ lily chain state-inspect`
        * `$ lily chain state-compute-range`
        * `$ lily chain state-compute`
* command to print actor CID's and version [#1026](https://github.com/filecoin-project/lily/pull/1026)
  * command: `$ lily chain actor-codes`
* tipSetWorker accepts all storage systems [#1035](https://github.com/filecoin-project/lily/pull/1035)
* update docker-compose to use lily worker pattern [#1044](https://github.com/filecoin-project/lily/pull/1044)


### Fix
* extract miner sector state changes [#1058](https://github.com/filecoin-project/lily/pull/1032)
  * improves accuracy of `miner_sector_events` task
* replace ExecutedAndBlockMessages with individual methods [#1040](https://github.com/filecoin-project/lily/pull/1040)
  * improves accuracy and performance of all message and receipt related tasks

### Chore
* lotus v1.17.0 [#1037](https://github.com/filecoin-project/lily/pull/1037)
* upgrade docker engine [#1047](https://github.com/filecoin-project/lily/pull/1047)
* removed unused gas calc from parsedmessages [#1056](https://github.com/filecoin-project/lily/pull/1056)
* use web3.storage gateway [#1051](https://github.com/filecoin-project/lily/pull/1051)
* prevent MinerSectorEvent panic on empty SectorStateEvents [#1053](https://github.com/filecoin-project/lily/pull/1053)
* handle vm message param and return parsing [#1057](https://github.com/filecoin-project/lily/pull/1057)
* correct column comment on vm_messages [#1058](https://github.com/filecoin-project/lily/pull/1058)

<a name="v0.11.0"></a>
## [v0.11.0] - 2022-06-05


## DATABASE MIGRATION

This release includes a migration which must be applied before deploying against an existing database.

### Feat
* support network version 16 [#988](https://github.com/filecoin-project/lily/pull/988)
* add is_string column to market_deal_proposal model [#1015](https://github.com/filecoin-project/lily/pull/1015)

### Fix
* prevent panic on parent miner state nil [#1013](https://github.com/filecoin-project/lily/pull/1013)
* allow tasks to be retired with block not found [#1016](https://github.com/filecoin-project/lily/pull/1016)
* many incorrect pointer equality comparisons [#1018](https://github.com/filecoin-project/lily/pull/1018)

### Chore
* implement miner sector event extractor tests [#999](https://github.com/filecoin-project/lily/pull/999)
* add output flag to sync status command [#1010](https://github.com/filecoin-project/lily/pull/1010)



<a name="v0.10.1"></a>
## [v0.10.1] - 2022-05-23

# IMPORTANT!

This release fixes a bug which causes data to be lost. This bug was introduced in `v0.9.0` and may require special attention around production of certain `miner`-based datasets. You can find details of the bug in [this issue](https://github.com/filecoin-project/lily/issues/997) along with their associated fixes. If you rely on your own Lily instances _AND HAVE DEPLOYED_ Lily `v0.9.0`, we recommend updating to `v0.10.1` or later. With `v0.10.1`, you may lily walk the epochs which were produced by `v0.9.0` and Lily will fill in the missing data.

The following datasets are affected:
- `miner_sector_infos_v7`
- `miner_sector_infos`
- `miner_sector_deals`
- `miner_locked_funds`

### Fix

- `lily job wait` respects Ctrl-C ([#976](https://github.com/filecoin-project/lily/pull/976))
- Register Miner Locked Funds ([#996](https://github.com/filecoin-project/lily/pull/996))
- Ensure redis queue inspector is registered for metric export ([#1011](https://github.com/filecoin-project/lily/pull/1011))

### Chore

- Update dependencies
  - lotus and FFI ([#973](https://github.com/filecoin-project/lily/pull/973))
  - go-amt-ipld ([#974](https://github.com/filecoin-project/lily/pull/974))
- Reject invalid tasks on client during job creation ([#975](https://github.com/filecoin-project/lily/pull/975))
- Improve distributed indexing metrics ([#981](https://github.com/filecoin-project/lily/pull/981))
- Fix `make test` ([#995](https://github.com/filecoin-project/lily/pull/995))
- Index manager unit tests ([#982](https://github.com/filecoin-project/lily/pull/982))
- Unify dev and CI linting ([#990](https://github.com/filecoin-project/lily/pull/990))
- Implement unit tests for Miner Deal extraction ([#994](https://github.com/filecoin-project/lily/pull/994))
- Implement unit tests for Tasktype and state processor lists ([#998](https://github.com/filecoin-project/lily/pull/998))
- Walk test uses in memory storage ([#1000](https://github.com/filecoin-project/lily/pull/1000))
- Increase resource_class for CI test ([#1003](https://github.com/filecoin-project/lily/pull/1003))

<a name="v0.10.0"></a>
## [v0.10.0] - 2022-05-23

### Feat

- Distributed Worker Pattern (#929)
  - Implementation of pattern described in [Distributed Lily Worker Pattern Doc](https://www.notion.so/pl-strflt/Distributed-Lily-Worker-Pattern-bbf101e392144e4184fae1ef7074ee02)
  - Modifies Lily to distribute tipset indexing across a pool of lily nodes using a redis queue. Lily may now run as a tipset-worker or a notifier
    - Notifier
      - Lily will add tipsets for indexing to a redis queue, walk, watch, fill, and index jobs may be used to notify the queue of tipsets to index.
    - Tipset-Worker
      - Lily will consume tipsets from a redis queue and index their state.
- Improved Job CLI (#944)
  - All jobs: `watch`, `walk`, `index`, `find`, `fill`, and `survey` are now under the `job run` command.
  - Improved documentation describing how to run each type of job and the work it performs.
  - Extend `watch`, `walk`, `index` and `fill` jobs with a sub-command `notify` which causes the corresponding job to notify configured redis queue of work to process.
- Parallel HAMT Diffing (#952)
  - Allows HAMTs to be diffed in parallel with a configurable number of goroutines, improving indexing performance.
- Parallel AMT Diffing (#953)
  - Allows AMTs to be diffed in parallel with a configurable number of goroutines, improving indexing performance.

### Fix
- Include parsed_message task in alias (#951)
  - includes the `parsed_message` task in the `messages` task alias. 
- Update urfave/cli to the latest version (#965)
  - fixes #964 ensuring task names are properly parsed on the CLI.
- Remove null characters from market_deal_proposal label field (#957)
  - fixes #930 by preventing invalid data from being written to CSV files for the market_deal_proposal model

### Chore
- Adjust help models-list width (#959)
- Update to lotus@v1.15.2 by (#954)


<a name="v0.9.0"></a>
## [v0.9.0] - 2022-04-21

### Feat

- Index Command (#871)

   - Lily received a new command - **index** - The index command can be used to index a single tipset, referenced by either its height or its TipSetKey. This command will block until indexing the tipset has completed.
   ```bash
   NAME:
      lily index - Index the state of a tipset from the filecoin blockchain.
    
   USAGE:
      lily index command [command options] [arguments...]
    
   COMMANDS:
      tipset   Index the state of a tipset from the filecoin blockchain by tipset key
      height   Index the state of a tipset from the filecoin blockchain by height
      help, h  Shows a list of commands or help for one command
    
   OPTIONS:
      --tasks value      Comma separated list of tasks to run. Each task is reported separately in the database. [$LILY_TASKS]
      --storage value    Name of storage that results will be written to. [$LILY_STORAGE]
      --api value        Address of lily api in multiaddr format. (default: "/ip4/127.0.0.1/tcp/1231") [$LILY_API]
      --api-token value  Authentication token for lily api. [$LILY_API_TOKEN]
      --name value       Name of job for easy identification later. [$LILY_JOB_NAME]
      --window value     Duration after which any indexing work not completed will be marked incomplete (default: 30s) [$LILY_WINDOW]
      --help, -h         show help (default: false)
   ```
  
- Granular Task Management (#871)
  - Tasks in Lily now correspond to the tables they produce. Users of Lily can specify the exact models they would like Lily to extract as opposed to previous behavior that grouped sets of models under a singe task name. See the below table for a mapping of task names to models they extract, and note that previous task names (📃) now map to the set of tasks Lily will run.
  - **NOTE: This requires that operators call `gap find` differently depending on whether they target the heights before or after the upgrade epoch (using the legacy task names in --tasks in the old epochs and using the granular task names in the newer epochs). Operators could optionally create a view which makes the legacy processing reports compatible with the new `gap find` query.**
  - Lily Task Table

    | Task Name | Model(s) |
    |-----------|----------|
    |📃 actorstatesinit | id_addresses |
    |📃 actorstatesmarket | market_deal_proposals, market_deal_states |
    |📃 actorstatesminer|miner_sector_deals, miner_sector_infos_v7, miner_sector_infos, miner_sector_posts, miner_pre_commit_infos, miner_sector_infos, miner_sector_events, miner_current_deadline_infos, miner_fee_debt, miner_locked_funds, miner_infos|
    |📃 actorstatesmultisig|multisig_transactions|
    |📃 actorstatespower|chain_powers, power_actor_claims|
    |📃 actorstatesraw|actors, actor_states|
    |📃 actorstatesreward|chain_rewards|
    |📃 actorstatesverifreg|verified_registry_verified_clients, verified_registry_verifiers|
    |📃 blocks|block_headers, block_parents, drand_block_entires|
    |📃 chaineconomics|chain_economics|
    |📃 consensus|chain_consensus|
    |📃 implicitmessage|internal_messages, internal_parsed_messages|
    |📃 messages|messages, block_messages, receipts, message_gas_economy, derived_gas_outputs|
    |📃 msapprovals|multisig_approvals|
    | actor|actors|
    | actor_state|actor_states|
    | block_header|block_headers|
    | block_message|block_messages|
    | block_parent|block_parents|
    | chain_consensus|chain_consensus|
    | chain_economics|chain_economics|
    | chain_power|chain_powers|
    | chain_reward|chain_rewards|
    | derived_gas_outputs|derived_gas_outputs|
    | drand_block_entrie|drand_block_entires|
    | id_address|id_addresses|
    | internal_messages|internal_messages|
    | internal_parsed_messages|internal_parsed_messages|
    | market_deal_proposal|market_deal_proposals|
    | market_deal_state|market_deal_states|
    | message|messages|
    | message_gas_economy|message_gas_economy|
    | miner_current_deadline_info|miner_current_deadlines|
    | miner_fee_debt|miner_fee_debts|
    | miner_info|miner_infos|
    | miner_locked_fund|miner_locked_funds|
    | miner_pre_commit_info|miner_pre_commit_infos|
    | miner_sector_deal|miner_sector_deals|
    | miner_sector_event|miner_sector_events|
    | miner_sector_infos|miner_sector_infos|
    | miner_sector_infos_v7|miner_sector_infos_v7|
    | miner_sector_post|miner_sector_posts|
    | multisig_approvals|multisig_approvals|
    | multisig_transaction|multisig_transactions|
    | parsed_message|parsed_messages|
    | power_actor_claim|power_actor_claims|
    | receipt|receipts|
    | verified_registry_verified_client|verified_registry_clients|
    | verified_registry_verifier|verified_registry_verifier|
    As an example, the command:
    ```bash
    $ lily index height 1741960 --tasks=actorstatesraw,actorstatespower,actorstatesreward,actorstatesminer,actorstatesinit,actorstatesmarket,actorstatesmultisig,actorstatesverifreg,block,messages,chaineconomics,msapprovals,implicitmessage,consensus
    ```
    will produce the follow entries in the processing_reports table

    | height | task | status |
    | --- | --- | --- |
    | 1741960 | verified_registry_verifier | OK |
    | 1741960 | verified_registry_verified_client | OK |
    | 1741960 | receipt | OK |
    | 1741960 | power_actor_claim | OK |
    | 1741960 | parsed_message | OK |
    | 1741960 | multisig_transaction | OK |
    | 1741960 | multisig_approvals | OK |
    | 1741960 | miner_sector_post | OK |
    | 1741960 | miner_sector_infos_v7 | OK |
    | 1741960 | miner_sector_infos | OK |
    | 1741960 | miner_sector_event | OK |
    | 1741960 | miner_sector_deal | OK |
    | 1741960 | miner_pre_commit_info | OK |
    | 1741960 | miner_locked_fund | OK |
    | 1741960 | miner_info | OK |
    | 1741960 | miner_fee_debt | OK |
    | 1741960 | miner_current_deadline_info | OK |
    | 1741960 | message_gas_economy | OK |
    | 1741960 | message | OK |
    | 1741960 | market_deal_state | OK |
    | 1741960 | market_deal_proposal | OK |
    | 1741960 | internal_parsed_messages | OK |
    | 1741960 | internal_messages | OK |
    | 1741960 | id_address | OK |
    | 1741960 | drand_block_entrie | OK |
    | 1741960 | derived_gas_outputs | OK |
    | 1741960 | chain_reward | OK |
    | 1741960 | chain_power | OK |
    | 1741960 | chain_economics | OK |
    | 1741960 | chain_consensus | OK |
    | 1741960 | builtin | OK |
    | 1741960 | block_parent | OK |
    | 1741960 | block_message | OK |
    | 1741960 | block_header | OK |
    | 1741960 | actor_state | OK |
    | 1741960 | actor | OK |

- Watcher Workers (#871)
  - The `lily watch` command now has the option to specify the number of tipsets that may be indexed in parallel. This can be specified with the `--workers` flag (see `lily watch --help` for details). Note that using large values (greater than 2) will require significant resource to coupe with indexing demands.

- Blockstore and Statestore Caching (#766)
  - The `lily daemon` command now as the option to use a `blockstore` and/or `statestore` cache. The `blockstore` cache caches values from the lotus blockstore, the `statestore` cache caches values from the lotus statestore. Setting these flags will increase the performance of the lily daemon, but will cause lily to use more memory. Assuming lily is operating on a system with 256 GiB of RAM we recommend the following values: `--blockstore-cache-size=5000000` and `--statestore-cache-size=3000000`. Some tuning will be required to find the values that work best for your deployment.

### Fix

- Gap Find (#884)
  - Gap find now exists as a single SQL function - requiring a migration to schema version 1.6. This function improves the performance and **accuracy** of the `gap find` command.

- Stateless Indexer (#871)
  - Lilys indexer is now stateless meaning the error `mismatching current and next tipsets` will no longer happen while indexing tipsets.

- Write Task Errors to Database (#871)
  - When tasks error their error message is correctly written to the `visor_processing_reports` table.

- Watch is tolerant of chain re-orgs (#871)
  - The watcher can now correctly watch the chain when a reorg occurs.

- Include job name in TipSetHeight metric (#918)
  - Allows metric collection on each jobs processing height.

- Prevent lily from running with incompatible schema (#910)
  - Ensures lily job only execute against correct schema version

<a name="v0.8.8"></a>
## [v0.8.8] - 2022-03-11

### Fix
- state cache falls through to slow path on error (#885)
- track dealID's of snapped sector (#901)
- return error when to and from are invalid (#906)
- Rollback v15 ProveCommitAggregate param schema to match (#907)

### Chore
- Adjust comments; Fix autodeploy regexs (#895)
- Dockerfiles are no longer included in make clean (#896)

### NOTICE: Fix for data bug affecting processing from Lily since v0.8.6

Fixes in v0.8.8 ([Fix 1](https://github.com/filecoin-project/lily/pull/901), [Fix 2](https://github.com/filecoin-project/lily/pull/907)) adjust data which has occurred in the past (before this release). In order to repair data produced by v0.8.6 and fixed in v0.8.8, please follow these steps:

1. Upgrade all lily instances which are writing to the database to be repaired to v0.8.8 (or later).
2. Run the following SQL to remove any data which may have been incorrectly written.

    ```sql
    -- parsed_messages may require the schema name for it to work properly: DELETE FROM customschemaname.parsed_messages WHERE...

    DELETE FROM parsed_messages
    WHERE height > 1594679;
    ```

3. Start a new job in lily to reprocess the affected heights:

    ```sh
    $ lily walk --name repair-data-0.8.8 --from 1594679 --to <CURRENTHEIGHT> --tasks=messages,actorstatesminer
    ```

4. Observe the processing reports and job results to ensure the walk completes successfully.

    ```sql
    SELECT FROM visor_processing_reports WHERE reporter = 'repair-data-0.8.8';
    ```

    ```sh
    $ lily job list
    ```

These repair steps will add data into `miner_sector_deals` for Snapped deals which were added in the v15 network upgrade and will repopulate the removed `parsed_messages` data which had unintended changes to the JSON schema in the `params` column of that table.

If there are any questions or problems, please [open a support issue](https://github.com/filecoin-project/lily/issues/new/choose).


<a name="v0.8.7"></a>
## [v0.8.7] - 2022-03-07

### Fix
- avoid custom marshal for zero and invalid values (#893)
- wrong use of fallthrough in marshalling paramWrapperType (#890)
- avoid panic on reflected param parsing code (#888)

<a name="v0.8.6"></a>
## [v0.8.6] - 2022-02-25

### Feat
- add block and actor caches (#766)
- set log level with regular expression (#876)
- job wait api (#785)
- implement v1.5 schema (#823)

### Fix
- ignore message parse errors for failed messages with unknown actor addresses (#806)
- panic due to res.Task being nil for skips (#814)
- add integration test build flag (#827)
- pass correct tipset to getTransactionIfApplied (#829)
- daemon help and config load (#831)
- dirty docker builds are now clean (#835)
- sanitize illegal utf in market deal proposal labels (#844)
- update docker base image for prod; Fix libc mismatch (#848)
- add test to ensure docker images are executable (#848)
- error marshalling in processing reports (#855)
- continuous deployment for tags and deploy branches (#868)
- persisting miner task results with nil models (#882)

### Chore
- improve tracing data and include lotus tracing (#794)
- replace ported state diff with lotus actor registry (#799)
- generic in memory lily vector test-suite (#811)
- implement wrapper type for custom message param json marshalling (#821)
- update to go1.17 (#822)
- Update to lotus 1.14.1 & specs-actors 7.0.0 (#823)
- add InternalParsedMessage to schema compatibility checklist (#833)
- add method to get csv headers for model (#834)
- set default log level to info (#870)
- an error when --to and --from are reversed (#872)
- Document --bootstrap flag in daemon command (#873)

<a name="v0.8.5"></a>
## [v0.8.5] - 2021-12-02

### Fix
- ci and testing deadlines (#776)
- message task tolerates unexpected params (#780)
- don't report errors for messages with invalid method (#784)
- ensembleminimal requires mocker proofs (#790)
- consider partially completed heights w ERRORs and SKIPs (#791)

### Chore
- bump go and timescale versions for CI tasks (#777)
- consolidate logger names (#789)

<a name="v0.8.4"></a>
## [v0.8.4] - 2021-10-29

### Fix
- msapprovals task supports actors v5 and above (#763)

<a name="v0.8.3"></a>
## [v0.8.3] - 2021-10-25

### Fix
- revert the correct tipset (#760)

<a name="v0.8.2"></a>
## [v0.8.2] - 2021-10-25

### Feat
- add observe command with peer agents task (#755)
- implement TipSetCache warming (#739)
- lily chain export command (#744)

### Fix
- watcher respects window timeout value (#710)
- job list sorted order (#731)
- return non-zero exit code on cli error (#732)
- normalize output upon job creation (#733)
- ensure all tasks are in task list (#736)
- daemon api uses full flag value (#748)
- populate event field of VerifiedRegistryVerifiedClient (#757)

### Chore
- wrap error from failed execution traces (#725)
- Create latest tags when pushing semver docker images (#728)
- Include jq in docker images; Fix dev entrypoint (#729)
- lotus 1.13 (#734)
- update readme linking to the documentation website. (#746)
- add lily stub/options for docker-compose (#747)
- update otel v0.20.0 and pg/v10 v10.10.6 (#749)
- implement chain set head (#750)
- update message parsing to use latest statediff generator (#756)
- improve logging around gapfill (#758)

<a name="v0.8.1"></a>
## [v0.8.1] - 2021-09-23

### Feat
 - add environment variables for command options (#716)

### Fix
 - implement method table for account actor (#714)
 - record internal message start and end time (#712)
 - consider initialization when deciding to create base schema (#707)

### Chore
 - issue template uses lily (#713)

<a name="v0.8.0"></a>
## [v0.8.0] - 2021-08-27

Note: This is a major release. The project name has changed from `sentinel-visor` to `lily`, the binary
produced when building the project has changed from `visor` to `lily`, and all lenses have been removed.
This means lily will cease to work with the lotusAPI lens, CAR-file lens, and Repo lens. Henceforth, Lily
will run as a deamonized process capable of both syncing and processing the Filecoin blockchain.

### Docs
- strengthen constraints on schema migrations (#683)

### Chore
- remove unused lens implementations (#654)
- uncomment skip logic in TestFind (#682)
- rename sentinel-visor/visor -> lily (#677)
- remove v0 schema (#684)
- update CI w Timescale v2.x (#686)

<a name="v0.7.7"></a>
## [v0.7.7] - 2021-08-24

### Feat
- GapFind and GapFill task implementation (#598)

<a name="v0.7.6"></a>
## [v0.7.6] - 2021-08-24

### Feat
- implement consensus task (#615)
- revamp help command and add topic based help text (#546)
- implement verifreg actor tracking (#539)

### Fix
- report separate metric for actor state extraction (#633)
- replace current epoch estimation with new function (#631)
- add missing models to schema verification (#643)
- internal message bugs (#639)
- avoid missing block when miner loads receipt (#648)
- record correct height for messages and block_messages (#650)
- consistent processing report heights (#653)
- avoid reporting processing error for malformed messages (#671)
- avoid reading state of deleted actors (#672)

### Chore
- remove unused finality field (#632)
- track start and end times for jobs (#638)
- implement chain reading methods (#657)
- include window and storage in job params output (#630)

<a name="v0.7.5"></a>
## [v0.7.5] - 2021-08-09

### Feat
- Add option to omit writing csv headers (#617)
- Add basic templating of csv filenames (#622)

### Fix
- Only process message executions for tipsets on same chain (#618)
- Correctly handle error for internal message processing (#627)
- Use schema config for migrations (#628)

### Chore
- Log timings for message extraction (#626)
- Add newlines to walk and watch output, print command errors to stdout (#625)
- Use lowercase for id option for consistency (#623)
- Make from and to options required for walk command (#624)

<a name="v0.7.4"></a>
## [v0.7.4] - 2021-08-02

### Feat
 - Implement internal message task (#533)

### Fix
 - Export watch_height metric (#606)
 - Remove space in lint comment (#607) 

<a name="v0.7.3"></a>
## [v0.7.3] - 2021-08-02

### Feat
 - Add v1 schema migration capability (#578)
 - Add metrics for tipset cache used by watches (#599)
 - Upgrade to lotus@v1.11.0 (#601)

<a name="v0.7.2"></a>
## [v0.7.2] - 2021-07-02

### Fix
 - Skip actor and message tasks if tipset is not direct child of parent
 - Change failure metrics to counts and export tipset_skip metric 

### Feat
 - Add metrics for job lifecycle and model persistence

### Build
 - Cache vectors in CI

<a name="v0.7.1"></a>
## [v0.7.1] - 2021-06-29

### Fix
 - Avoid indexer blocking head events
 - Add batch persist support for v0 schema message tables
 - Increase API cache size 


<a name="v0.7.0"></a>
## [v0.7.0] - 2021-06-24

This version introduces a revised database schema, support for filecoin network version 13 and actors 5.

### Fix
 - Add usage for --api flag
 - Update amt package with diffing fix
 - Bare migrate command uses supplied schema name
 - Use actorname function generated for v5 actors
 - Better errors, equals method, correct cid
 - Give head notifier a buffer
 - Correctly handle the genesis state
 - log-level option should apply to all loggers
 - Add hamt map opts for v5 actors
 - Use correct log level regex
 - Ensure id_addresses contains height

### Feat
 - Introduce major schema versioning and version 1 schema
 - Add stop command
 - Add net commands for inspecting daemon connectivity
 - Decode message bitfields to json


### Other
 - Update lotus dependency to include v1.10 changes
 - Add support for v5 actors
 - Automated Calibnet Docker Builds
 - Removed unused visor processing models
 - Support building visor for devnets
 - Sort list of log levels
 - Replace golangci-lint by staticcheck
 - Support building and executing on M1-based Macs


<a name="v0.6.7"></a>
## [v0.6.7] - 2021-06-10

### Fix
 - Restart watcher on failure in daemon mode
 - Update statediff to include v4 miners
 - Parse plain value transfer messages
 - Fall back to using config in repo directory if unspecified
 - Expand the config path when starting daemon 

### Feat
 - Add logging command and sane default log levels

### Chore
 - Remove s3 lens
 - Remove dependency on statediff
 - Update to use latest version of Lotus

<a name="v0.6.6"></a>
## [v0.6.6] - 2021-05-28

### Fix
 - Track miner sector size

### Feat
 - Introduce major schema versioning
 - Upgrade HAMT and AMT diffing to more efficient implementation when able to
 - Implement optimized state tree diffing

### Chore
 - Add actor-shim code generation
 - Add more tracing calls to critial path


<a name="v0.6.5"></a>
## [v0.6.5] - 2021-05-20

### Change
 - fail parsed message task when an unknown method is encountered

### Fix
 - update statediff to v0.0.24 to properly support actor v4 message methods and parameters

<a name="v0.6.4"></a>
## [v0.6.4] - 2021-05-19

### Chore
 - Update Lotus dependency to latest master to include v1.9.0 changes

<a name="v0.6.3"></a>
## [v0.6.3] - 2021-05-18

### Fix
 - setup logging, metrics and tracing in daemon
 - import correct v4 actor package for msapprovals 
 - jsonrpc only supports returning 2 params 
 - message task tracks unexecuted messages and their blocks
 - only exit once all scheduled jobs complete 
 - allow scheduler to exit if scheduled jobs are complete 

### Feat
 - allow database urls to use environment variables in config
 - add wait-api command to wait for the visor api to come online
 - add init command

### Chore
 - shorten indexhead-confidence option name on watch command
 - fix typos in init command usage


<a name="v0.6.2"></a>
## [v0.6.2] - 2021-04-28

### Fix
 - improve reliability of reconnecting to lotus after disconnect

### Feat
 - Upgrade lotus dependency to v1.8.0
 - Import more lotus commands

### Perf
 - Remove redundant tipset lookups

### Chore
 - Download vectors before running tests in CI


<a name="v0.6.1"></a>
## [v0.6.1] - 2021-04-21

### Feat
 - Upgrade actor support to version 4

### Chore
 - Fix dev docker image
 - Add circleci check for docker builds


<a name="v0.6.0"></a>
## [v0.6.0] - 2021-04-20

### BREAKING CHANGE

This changes the cli interface to make `walk` and `watch` subcommands of a new `run` command. Command line options
that are specific to `walk` and `watch` must now be specified after the subcommand.

Where before a walk would be started like this:

```sh
$ visor --db=foo --lens=lotus walk --from=1000 --to=1001
```

it must now be started like this:

```sh
$ visor run walk --db=foo --lens=lotus --from=1000 --to=1001
```

And for `watch`, before:

```sh
$ visor --db=foo --lens=lotus watch --headindexer-confidence=100
```

after:

```sh
$ visor run watch --db=foo --lens=lotus --headindexer-confidence=100
```

The `migrate` command now also expects options to be specified after the command:

```sh
$ visor migrate --db=foo --latest
```


In addition: 

 - the `--api` option has been renamed to `--lens-lotus-api` since it is used by the lotus lens to specify the api that visor will connect to. This reduces confusion from the `daemon` command's `--api` option which specifies the api address of visor's daemon.
 - the `--repo` option has been renamed to `--lens-repo` since it is used by various lenses to specify the location of the data file or directory that visor will read from. This distinguishes it from the `daemon` command's `--repo` option which specifies the path where the visor daemon should write its data.



### Feat
 - Add a new long running daemon mode for visor
 - Add new daemon command
 - Add new job command
 - Add new run command and move watch and walk to be subcommands of run
 - Reorganise CLI options to be associated with relevant command or subcommand

<a name="v0.5.7"></a>
## [v0.5.7] - 2021-04-09

### Fix
 - Ensure persistence semaphore channel is drained on close

<a name="v0.5.6"></a>
## [v0.5.6] - 2021-04-07

### Chore
 - update to Lotus 1.6.0


<a name="v0.5.5"></a>
## [v0.5.5] - 2021-03-25

### Fix
 - only close TIpSetIndexer in walk and watch
 - embed genesis block in executable
 - ignore false positive gosec failure in wait package
 - close TipSetObs in walker and watcher

### Feat
 - add benchmarking of vectors
 - record tipset cache metrics during watch
 - Support current sqlotus schema
 - sqlotus dag prefetch option

### Chore
 - update for Lotus 1.5.3
 - increase linter timout in ci
 - remove lint github action
 - fix the linter



<a name="v0.5.4"></a>
## [v0.5.4] - 2021-03-09

### Fix
- guard concurrent accesses to node api ([#412](https://github.com/filecoin-project/sentinel-visor/issues/412))
- avoid deadlock in indexer when processor errors ([#407](https://github.com/filecoin-project/sentinel-visor/issues/407))

<a name="v0.5.3"></a>
## [v0.5.3] - 2021-03-02

### Feat
- support Lotus 1.5.0 and Actors 3.0.3 ([#373](https://github.com/filecoin-project/sentinel-visor/issues/373))

### Fix
- wait for persist routines to complete in Close ([#374](https://github.com/filecoin-project/sentinel-visor/issues/374))

<a name="v0.5.2"></a>
## [v0.5.2] - 2021-02-22

### Feat
- record multisig approvals ([#389](https://github.com/filecoin-project/sentinel-visor/issues/389))
- implement test vector builder and executer ([#370](https://github.com/filecoin-project/sentinel-visor/issues/370))

### Fix
- msapprovals missing pending transaction ([#395](https://github.com/filecoin-project/sentinel-visor/issues/395))
- correct docker image name; simplify build pipeline ([#393](https://github.com/filecoin-project/sentinel-visor/issues/393))
- set chainstore on repo lens util

### Chore
- add release process details and workflow ([#353](https://github.com/filecoin-project/sentinel-visor/issues/353))

<a name="v0.5.1"></a>
## [v0.5.1] - 2021-02-09

### Feat
- record actor task metrics ([#376](https://github.com/filecoin-project/sentinel-visor/issues/376))

### Chore
- increase lens object cache size ([#377](https://github.com/filecoin-project/sentinel-visor/issues/377))

### Schema
- remove use of add_drop_chunks_policy timescale function ([#379](https://github.com/filecoin-project/sentinel-visor/issues/379))


<a name="v0.5.0"></a>
## [v0.5.0] - 2021-02-09

No changes from v0.5.0-rc2


<a name="v0.5.0-rc2"></a>
## [v0.5.0-rc2] - 2021-01-27

Required schema version: `27`

### Notable for this release:
- update specs-actors to support v3 upgrade
- CSV exporting for easier ingestion into the DB of your choice
- bug fix for incorrect gas outputs (which changed after FIP-0009 was applied)
- inline schema documentation

### Feat
- remove default value for --db parameter ([#348](https://github.com/filecoin-project/sentinel-visor/issues/348))
- abstract model storage and add csv output for walk command ([#316](https://github.com/filecoin-project/sentinel-visor/issues/316))
- allow finer-grained actor task processing ([#305](https://github.com/filecoin-project/sentinel-visor/issues/305))
- record metrics for watch and walk commands ([#312](https://github.com/filecoin-project/sentinel-visor/issues/312))
- **db:** allow model upsertion
- **gas outputs:** Add Height and ActorName ([#270](https://github.com/filecoin-project/sentinel-visor/issues/270))
- **lens:** Optimize StateGetActor calls. ([#214](https://github.com/filecoin-project/sentinel-visor/issues/214))

### Fix
- persist market deal states ([#367](https://github.com/filecoin-project/sentinel-visor/issues/367))
- improve inferred json encoding for csv output ([#364](https://github.com/filecoin-project/sentinel-visor/issues/364))
- csv output handles time and interface values ([#351](https://github.com/filecoin-project/sentinel-visor/issues/351))
- adjust calculation of gas outputs for FIP-0009 ([#356](https://github.com/filecoin-project/sentinel-visor/issues/356))
- reject names that exceed maximum postgres name length ([#323](https://github.com/filecoin-project/sentinel-visor/issues/323))
- don't restart a walk if it fails ([#320](https://github.com/filecoin-project/sentinel-visor/issues/320))
- close all processor connections to lotus on fatal error ([#309](https://github.com/filecoin-project/sentinel-visor/issues/309))
- use migration database connection when installing timescale extension ([#304](https://github.com/filecoin-project/sentinel-visor/issues/304))
- **ci:** Pin TimescaleDB to v1.7 on Postgres v12 ([#340](https://github.com/filecoin-project/sentinel-visor/issues/340))
- **migration:** don't recreate miner_sector_deals primary key if it is correct ([#300](https://github.com/filecoin-project/sentinel-visor/issues/300))
- **migrations:** CREATE EXTENSION deadlocks inside migrations global lock ([#210](https://github.com/filecoin-project/sentinel-visor/issues/210))
- **miner:** extract miner PoSt's from parent messages

### Chore
- update imports and ffi stub for lotus 1.5.0-pre1 ([#371](https://github.com/filecoin-project/sentinel-visor/issues/371))
- fix some linting issues ([#349](https://github.com/filecoin-project/sentinel-visor/issues/349))
- **api:** trim the lens API to required methods
- **lint:** fix linter errors
- **lint:** fix staticcheck linting issues ([#299](https://github.com/filecoin-project/sentinel-visor/issues/299))
- **sql:** user numeric type to represent numbers ([#327](https://github.com/filecoin-project/sentinel-visor/issues/327))

### Perf
- replace local state diffing with StateChangeActors API method ([#303](https://github.com/filecoin-project/sentinel-visor/issues/303))

### Test
- **actorstate:** unit test actorstate actor task
- **chain:** refactor and test chain economics extraction ([#298](https://github.com/filecoin-project/sentinel-visor/issues/298))

### Docs
- table and column comments ([#346](https://github.com/filecoin-project/sentinel-visor/issues/346))
- Update README and docker-compose to require use of TimescaleDB v1.7 ([#341](https://github.com/filecoin-project/sentinel-visor/issues/341))
- document mapping between tasks and tables ([#369](https://github.com/filecoin-project/sentinel-visor/issues/369))

### Polish
- **test:** allow `make test` to "just work"

## [v0.5.0-rc1] - Re-released as [v0.5.0-rc2](#v0.5.0-rc2)

<a name="v0.4.0"></a>
## [v0.4.0] - 2020-12-16
### Chore
- remove test branch and temp deploy config

### Fix
- Make visor the entrypoint for dev containers


<a name="v0.4.0-rc2"></a>
## [v0.4.0-rc2] - 2020-12-16
### Feat
- **ci:** Dockerfile.dev; Refactor docker push steps in circleci.yaml


<a name="v0.4.0-rc1"></a>
## [v0.4.0-rc1] - 2020-12-02
### DEPRECATION

The CLI interface has shifted again to deprecate the `run` subcommand in favor of dedicated subcommands for `indexer` and `processor` behaviors.

Previously the indexer and procerror would be started via:

```sh
  sentinel-visor run --indexhead
  sentinel-visor run --indexhistory
```

After this change:

```sh
  sentinel-visor watch
  sentinel-visor walk
```

The `run` subcommand will be removed in v0.5.0.

### Feat
- extract basic account actor states ([#278](https://github.com/filecoin-project/sentinel-visor/issues/278))
- add watch and walk commands to index chain during traversal ([#249](https://github.com/filecoin-project/sentinel-visor/issues/249))
- functions to convert unix epoch to fil epoch ([#252](https://github.com/filecoin-project/sentinel-visor/issues/252))
- add repo-read-only flag to enable read or write on lotus repo ([#250](https://github.com/filecoin-project/sentinel-visor/issues/250))
- allow application name to be passed in postgres connection url ([#243](https://github.com/filecoin-project/sentinel-visor/issues/243))
- limit history indexer by height ([#234](https://github.com/filecoin-project/sentinel-visor/issues/234))
- extract msig transaction hamt

### Fix
- optimisable height functions ([#268](https://github.com/filecoin-project/sentinel-visor/issues/268))
- don't update go modules when running make
- gracefully disconnect from postgres on exit
- truncated tables in tests ([#277](https://github.com/filecoin-project/sentinel-visor/issues/277))
- tests defer database cleanup without invoking ([#274](https://github.com/filecoin-project/sentinel-visor/issues/274))
- totalGasLimit and totalUniqueGasLimit are correct
- missed while closing [#201](https://github.com/filecoin-project/sentinel-visor/issues/201)
- include height with chain power results ([#255](https://github.com/filecoin-project/sentinel-visor/issues/255))
- avoid panic when miner has no peer id ([#254](https://github.com/filecoin-project/sentinel-visor/issues/254))
- Remove hack to RestartOnFailure
- Reorder migrations after merging latest master ([#248](https://github.com/filecoin-project/sentinel-visor/issues/248))
- multisig actor migration
- lotus chain store is a blockstore
- panic in multisig genesis task casting
- **actorstate:** adjust account extractor to conform to new interface ([#294](https://github.com/filecoin-project/sentinel-visor/issues/294))
- **init:** extract idAddress instead of actorID
- **schema:** fix primary key for miner_sector_deals table ([#291](https://github.com/filecoin-project/sentinel-visor/issues/291))

### Refactor
- **cmd:** Modify command line default parameters ([#271](https://github.com/filecoin-project/sentinel-visor/issues/271))

### Test
- add multisig actor extractor tests
- power actor claim extration test
- **init:** test coverage for init actor extractor

### Chore
- Avoid ingesting binary and unused data ([#241](https://github.com/filecoin-project/sentinel-visor/issues/241))
- remove unused tables and views

### CI
- **test:** add code coverage
- **test:** run full testing suite

### Build
- **ci:** add go mod tidy check ([#266](https://github.com/filecoin-project/sentinel-visor/issues/266))

### Docs
- expand getting started guide and add running tests section ([#275](https://github.com/filecoin-project/sentinel-visor/issues/275))

### Polish
- Avoid duplicate work when reading receipts
- use new init actor diffing logic
- **mockapi:** names reflect method action
- **mockapi:** remove returned errors and condense mockTipset
- **mockapi:** accepts testing.TB, no errors

<a name="v0.3.0"></a>
## [v0.3.0] - 2020-11-03
### Feat
- add visor processing stats table ([#96](https://github.com/filecoin-project/sentinel-visor/issues/96))
- allow actor state processor to run without leasing ([#178](https://github.com/filecoin-project/sentinel-visor/issues/178))
- rpc reconnection on failure ([#149](https://github.com/filecoin-project/sentinel-visor/issues/149))
- add dynamic panel creation based on tags ([#159](https://github.com/filecoin-project/sentinel-visor/issues/159))
- add dynamic panel creation based on tags
- make delay between tasks configurable ([#151](https://github.com/filecoin-project/sentinel-visor/issues/151))
- convert processing, block and message tables to hypertables ([#111](https://github.com/filecoin-project/sentinel-visor/issues/111))
- set default numbers of workers to zero in run subcommand ([#116](https://github.com/filecoin-project/sentinel-visor/issues/116))
- add dashboard for process completion
- add changelog generator
- log visor version on startup ([#117](https://github.com/filecoin-project/sentinel-visor/issues/117))
- Add heaviest chain materialized view ([#97](https://github.com/filecoin-project/sentinel-visor/issues/97))
- Add miner_sector_posts tracking of window posts ([#74](https://github.com/filecoin-project/sentinel-visor/issues/74))
- Add historical indexer metrics ([#92](https://github.com/filecoin-project/sentinel-visor/issues/92))
- add message gas economy processing
- set application name in postgres connection ([#104](https://github.com/filecoin-project/sentinel-visor/issues/104))
- **miner:** compute miner sector events
- **task:** add chain economics processing ([#94](https://github.com/filecoin-project/sentinel-visor/issues/94))

### Fix
- Make ChainVis into basic views
- failure to get lock when ExitOnFailure is true now exits
- use hash index type for visor_processing_actors_code_idx ([#106](https://github.com/filecoin-project/sentinel-visor/issues/106))
- fix actor completion query
- visor_processing_stats queries for Visor processing dash ([#156](https://github.com/filecoin-project/sentinel-visor/issues/156))
- remove errgrp from UnindexedBlockData persist
- migration table name
- correct typo in derived_consensus_chain_view name and add to view refresh ([#112](https://github.com/filecoin-project/sentinel-visor/issues/112))
- avoid panic when miner extractor does not find receipt ([#110](https://github.com/filecoin-project/sentinel-visor/issues/110))
- verify there are no missing migrations before migrating ([#89](https://github.com/filecoin-project/sentinel-visor/issues/89))
- **lens:** Include dependencies needed for Repo Lens ([#90](https://github.com/filecoin-project/sentinel-visor/issues/90))
- **metrics:** export the completion and batch selection views ([#197](https://github.com/filecoin-project/sentinel-visor/issues/197))
- **migration:** message gas economy uses bigint
- **migrations:** migrations require version 0
- **schema:** remove blocking processing indexes and improve processing stats table ([#130](https://github.com/filecoin-project/sentinel-visor/issues/130))

### Build
- add prometheus, grafana and dashboard images

### Chore
- Incl migration in CI test
- Include RC releases in push docker images ([#195](https://github.com/filecoin-project/sentinel-visor/issues/195))
- add metrics to leasing and work completion queries
- add changelog ([#150](https://github.com/filecoin-project/sentinel-visor/issues/150))
- update go.mod after recent merge ([#155](https://github.com/filecoin-project/sentinel-visor/issues/155))
- add issue templates
- add more error context reporting in messages task ([#133](https://github.com/filecoin-project/sentinel-visor/issues/133))

### Deps
- remove unused docker file for redis

### Perf
- ensure processing updates always include height in criteria ([#192](https://github.com/filecoin-project/sentinel-visor/issues/192))
- include height restrictions in update clauses of leasing queries ([#189](https://github.com/filecoin-project/sentinel-visor/issues/189))
- **db:** reduce batch size for chain history indexer ([#105](https://github.com/filecoin-project/sentinel-visor/issues/105))

### Polish
- update miner processing logic

### Test
- ensure docker-compose down runs on test fail


<a name="v0.2.0"></a>
## [v0.2.0] - 2020-10-11
### BREAKING CHANGE

this changes the cli interface to remove the run subcommand.

Previously the indexer and procerror would be started via:

```sh
  sentinel-visor run indexer
  sentinel-visor run processor
```

After this change:

```sh
  sentinel-visor index
  sentinel-visor process
```

### Feat
- add standard build targets ([#18](https://github.com/filecoin-project/sentinel-visor/issues/18))
- add licenses and skeleton readme ([#5](https://github.com/filecoin-project/sentinel-visor/issues/5))
- instrument with tracing ([#15](https://github.com/filecoin-project/sentinel-visor/issues/15))
- add a configurable delay between task restarts ([#71](https://github.com/filecoin-project/sentinel-visor/issues/71))
- compute gas outputs ([#67](https://github.com/filecoin-project/sentinel-visor/issues/67))
- add tests for indexer ([#12](https://github.com/filecoin-project/sentinel-visor/issues/12))
- add schema migration capability ([#40](https://github.com/filecoin-project/sentinel-visor/issues/40))
- add LILY_TEST_DB environment variable to specify test database ([#35](https://github.com/filecoin-project/sentinel-visor/issues/35))
- respect log level flag and allow per logger levels ([#34](https://github.com/filecoin-project/sentinel-visor/issues/34))
- remove run subcommand and make index and process top level
- embed version number from build
- support v2 actor codes ([#84](https://github.com/filecoin-project/sentinel-visor/issues/84))
- add test for create schema ([#3](https://github.com/filecoin-project/sentinel-visor/issues/3))
- **api:** wrap lotus api and store with wrapper
- **debug:** Process actor by head without persistance ([#86](https://github.com/filecoin-project/sentinel-visor/issues/86))
- **genesis:** add task for processing genesis state
- **scheduler:** Refactor task scheduler impl ([#41](https://github.com/filecoin-project/sentinel-visor/issues/41))
- **task:** add actor, actor-state, and init actor processing ([#14](https://github.com/filecoin-project/sentinel-visor/issues/14))
- **task:** implement message processing task
- **task:** add market actor task
- **task:** add reward actor processing ([#16](https://github.com/filecoin-project/sentinel-visor/issues/16))
- **task:** add power actor processing task ([#11](https://github.com/filecoin-project/sentinel-visor/issues/11))
- **task:** Create chainvis views and refresher ([#77](https://github.com/filecoin-project/sentinel-visor/issues/77))

### Fix
- use debugf logging method in message processor ([#82](https://github.com/filecoin-project/sentinel-visor/issues/82))
- chain history indexer includes genesis ([#72](https://github.com/filecoin-project/sentinel-visor/issues/72))
- use context deadlines only if task has been assigned work ([#70](https://github.com/filecoin-project/sentinel-visor/issues/70))
- fix failing chain head indexer tests ([#66](https://github.com/filecoin-project/sentinel-visor/issues/66))
- add migration to remove old chainwatch schema constraints ([#48](https://github.com/filecoin-project/sentinel-visor/issues/48))
- use noop tracer when tracing disabled ([#39](https://github.com/filecoin-project/sentinel-visor/issues/39))
- ensure processor stops scheduler when exiting ([#24](https://github.com/filecoin-project/sentinel-visor/issues/24))
- **build:** ensure deps are built befor visor
- **indexer:** don't error on empty blocks_synced table
- **model:** replace BeginContext with RunInTransaction ([#7](https://github.com/filecoin-project/sentinel-visor/issues/7))
- **task:** correct index when computing deal state
### Chore
- add tests for reward and power actor state extracters ([#83](https://github.com/filecoin-project/sentinel-visor/issues/83))
- fail database tests if LILY_TEST_DB not set ([#79](https://github.com/filecoin-project/sentinel-visor/issues/79))
- use clock package for time mocking ([#65](https://github.com/filecoin-project/sentinel-visor/issues/65))
- remove unused redis-based scheduler code ([#64](https://github.com/filecoin-project/sentinel-visor/issues/64))
- Push docker images on [a-z]*-master branch updates ([#49](https://github.com/filecoin-project/sentinel-visor/issues/49))
- Remove sentinel prefix for local dev use ([#36](https://github.com/filecoin-project/sentinel-visor/issues/36))
- push docker tags from ci ([#26](https://github.com/filecoin-project/sentinel-visor/issues/26))
- tighten up error propagation ([#23](https://github.com/filecoin-project/sentinel-visor/issues/23))
- fix docker hub submodule error ([#22](https://github.com/filecoin-project/sentinel-visor/issues/22))
- add circle ci ([#20](https://github.com/filecoin-project/sentinel-visor/issues/20))
- add docker build and make targets ([#19](https://github.com/filecoin-project/sentinel-visor/issues/19))

### Dep
- add fil-blst submodule

### Perf
- minor optimization of market actor diffing ([#78](https://github.com/filecoin-project/sentinel-visor/issues/78))
- use batched inserts for models ([#73](https://github.com/filecoin-project/sentinel-visor/issues/73))

### Pg
- configurable pool size

### Polish
- **processor:** parallelize actor change collection
- **publisher:** receive publish operations on channel
- **redis:** configure redis with env vars ([#21](https://github.com/filecoin-project/sentinel-visor/issues/21))

### Refactor
- prepare for specs-actors upgrade
- replace panic with error return in indexer.Start ([#4](https://github.com/filecoin-project/sentinel-visor/issues/4))

### Test
- **storage:** add test to check for duplicate schema migrations ([#80](https://github.com/filecoin-project/sentinel-visor/issues/80))

[v0.5.4]: https://github.com/filecoin-project/sentinel-visor/compare/v0.5.3...v0.5.4
[v0.5.3]: https://github.com/filecoin-project/sentinel-visor/compare/v0.5.2...v0.5.3
[v0.5.2]: https://github.com/filecoin-project/sentinel-visor/compare/v0.5.1...v0.5.2
[v0.5.1]: https://github.com/filecoin-project/sentinel-visor/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/filecoin-project/sentinel-visor/compare/v0.5.0-rc2...v0.5.0
[v0.5.0-rc2]: https://github.com/filecoin-project/sentinel-visor/compare/v0.4.0...v0.5.0-rc1
[v0.4.0]: https://github.com/filecoin-project/sentinel-visor/compare/v0.4.0-rc2...v0.4.0
[v0.4.0-rc2]: https://github.com/filecoin-project/sentinel-visor/compare/v0.4.0-rc1...v0.4.0-rc2
[v0.4.0-rc1]: https://github.com/filecoin-project/sentinel-visor/compare/v0.3.0...v0.4.0-rc1
[v0.3.0]: https://github.com/filecoin-project/sentinel-visor/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/filecoin-project/sentinel-visor/compare/b7044af...v0.2.0
