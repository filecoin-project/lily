# Changelog
All notable changes to this project will be documented in this file.

The format is a variant of [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) combined
with categories from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). Breaking changes 
should trigger an increment to the major version. Features increment the minor version and fixes or
other changes increment the patch number.


## [Unreleased]

These are changes that will probably be included in the next release.

### Breaking changes

### Features
 - add sql lens (#132) (2020-10-26)
 - make parseMsg task opt-in (#143) (2020-10-26)
 - add 30s bucket to metrics distribution as that is our block time (#115) (2020-10-26)
 - add car repo lens (#138) (2020-10-23)
 - bump statediff to v0.0.7 (#136) (2020-10-22)
 - feat: convert processing, block and message tables to hypertables (#111) (2020-10-22)
 - feat: set default numbers of workers to zero in run subcommand (#116) (2020-10-22)
 - feat: add dashboard for process completion (2020-10-16) 
 - include height with parsed messages (#121) (2020-10-19)
 - include schema with messages parsed in json field (#119) (2020-10-19)
 - test: ensure docker-compose down runs on test fail (2020-10-15) 
 - feat(miner): compute miner sector events (2020-10-08) 
 - feat: log visor version on startup (#117) (2020-10-16)
 - feat: Add heaviest chain materialized view (#97) (2020-10-14)
 - feat: Add miner_sector_posts tracking of window posts (#74) (2020-10-14)
 - feat: Add historical indexer metrics (#92) (2020-10-15)
 - Move migration to schema version 10 (2020-10-14) 
 - feat: add message gas economy processing (2020-10-13) 
 - feat: set application name in postgres connection (#104) (2020-10-14)
 - feat: add visor processing stats table (#96) (2020-10-14)
 - feat(task): add chain economics processing (#94) (2020-10-14)
 - get actor name for both versions of specs-actors (#101) (2020-10-14)

### Fixes
 - fix(schema): remove blocking processing indexes and improve processing stats table (#130) (2020-10-22)
 - Fix issues in repo lens and message parsing (#125) (2020-10-20)
 - fix: migration table name (2020-10-19) 
 - fix index creation syntax (#122) (2020-10-19)
 - fix(migration): message gas economy uses bigint (2020-10-14) 
 - fix: correct typo in derived_consensus_chain_view name and add to view refresh (#112) (2020-10-15)
 - fix: avoid panic when miner extractor does not find receipt (#110) (2020-10-15)
 - fix(lens): Include dependencies needed for Repo Lens (#90) (2020-10-14)
 - fix(migrations): migrations require version 0 (2020-10-13) 
 - fix: use hash index type for visor_processing_actors_code_idx (#106) (2020-10-14)
 - fix: remove errgrp from UnindexedBlockData persist (2020-10-13) 
 - fix: verify there are no missing migrations before migrating (#89) (2020-10-13)

### Other changes
 - chore: add more error context reporting in messages task (#133) (2020-10-22)
 - perf(db): reduce batch size for chain history indexer (#105) (2020-10-14)
 - deps: remove unused docker file for redis (2020-10-15) 
 - build: add prometheus, grafana and dashboard images (2020-10-15) 


## [v0.2.0] - 2020-10-11

First tagged release





