# Changelog
All notable changes to this project will be documented in this file.

The format is a variant of [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) combined with categories from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Breaking changes should trigger an increment to the major version. Features increment the minor version and fixes or other changes increment the patch number.


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
- add VISOR_TEST_DB environment variable to specify test database ([#35](https://github.com/filecoin-project/sentinel-visor/issues/35))
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
- fail database tests if VISOR_TEST_DB not set ([#79](https://github.com/filecoin-project/sentinel-visor/issues/79))
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

[v0.4.0]: https://github.com/filecoin-project/sentinel-visor/compare/v0.4.0-rc2...v0.4.0
[v0.4.0-rc2]: https://github.com/filecoin-project/sentinel-visor/compare/v0.4.0-rc1...v0.4.0-rc2
[v0.4.0-rc1]: https://github.com/filecoin-project/sentinel-visor/compare/v0.3.0...v0.4.0-rc1
[v0.3.0]: https://github.com/filecoin-project/sentinel-visor/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/filecoin-project/sentinel-visor/compare/b7044af...v0.2.0
