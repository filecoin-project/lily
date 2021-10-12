# Sentinel Visor
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/filecoin-project/lily) [![docker build status](https://img.shields.io/docker/cloud/build/filecoin/lily?style=flat-square)](https://hub.docker.com/repository/docker/filecoin/lily) [![CI build](https://img.shields.io/circleci/build/gh/filecoin-project/lily?label=ci%20build&style=flat-square)](https://app.circleci.com/pipelines/github/filecoin-project/lily)

A component of [**Sentinel**](https://github.com/filecoin-project/sentinel), a collection of services which monitor the health and function of the Filecoin network. 

A **Visor** process collects _permanent_ Filecoin chain metrics from a [**Lotus**](https://github.com/filecoin-project/lotus/) daemon, and writes them to a [**TimescaleDB**](https://github.com/timescale/timescaledb) time-series and relational datastore.

## Getting Started

Clone the repo and build the dependencies:

```console
$ git clone https://github.com/filecoin-project/lily
$ cd lily
$ make deps
```

Build the `lily` binary to the root of the project directory:

```console
$ make build
```

#### Building on M1-based Macs

Because of the novel architecture of the M1-based Mac computers, some specific environment variables must be set before creating the lily executable.

Create necessary environment variable to allow Visor to run on ARM architecture:
```console
export GOARCH=arm64
export CGO_ENABLED=1
export LIBRARY_PATH=/opt/homebrew/lib
export FFI_BUILD_FROM_SOURCE=1

```
Now, build the `lily` binary to the root of the project directory:

```console
$ make build
```

Install TimescaleDB v2.x:

In a separate shell, use docker-compose to start the appropriate version of Postgres with TimescaleDB.

```sh
docker-compose up --build timescaledb
```

### Running tests

To quickly run tests, you can provide the `LILY_TEST_DB` envvar and execute `make test` like so:

`LILY_TEST_DB="postgres://postgres:password@localhost:5432/postgres?sslmode=disable" make test`

For more, manual test running, you could also prepare your environment in the following way:

Create a new DB in postgres for testing:

```sql
CREATE DATABASE lily_test;
```

Migrate the database to the latest schema:

```sh
lily migrate --db "postgres://username@localhost/lily_test?sslmode=disable" --latest
```

Run the tests:

```sh
LILY_TEST_DB="postgres://username@localhost/lily_test?sslmode=disable" go test ./...
```

### Usage

```
  lily [<flags>] <command>

  Use 'lily help <command>' to learn more about each command.
```

Use the following env vars to configure the lotus node that lily reads from, and the database that it writes to:

- `LILY_PATH` - path to the lotus data dir. _default: `~/.lily`_
- `LILY_DB` - database connection . _default: `postgres://postgres:password@localhost:5432/postgres?sslmode=disable`_

The `walk` and `watch` commands expect a list of tasks to be provided. Each task is responsible for reading a particular type of data from the chain and persisting it to the database.
The mapping between available tasks and database tables is as follows:

| Task Name           | Database Tables |
|---------------------|-----------------|
| blocks              | block_headers, block_parents, drand_block_entries |
| messages            | messages, receipts, block_messages, parsed_messages, derived_gas_outputs, message_gas_economy |
| chaineconomics      | chain_economics |
| actorstatesraw      | actors, actor_states |
| actorstatespower    | chain_powers, power_actor_claims |
| actorstatesreward   | chain_rewards |
| actorstatesminer    | miner_current_deadline_infos, miner_fee_debts, miner_locked_funds, miner_infos, miner_sector_posts, miner_pre_commit_infos, miner_sector_infos, miner_sector_events, miner_sector_deals |
| actorstatesinit     | id_addresses |
| actorstatesmarket   | market_deal_proposals, market_deal_states |
| actorstatesmultisig | multisig_transactions |


### Configuring Tracing

The global flag `--tracing=<bool>` turns tracing on or off. It is on by default.

Tracing expects a Jaeger server to be available. Configure the Jaeger settings using the following subset of the standard Jaeger [environment variables](https://github.com/jaegertracing/jaeger-client-go#environment-variables):

 * `JAEGER_SERVICE_NAME` - name of the service (defaults to `lily`).
 * `JAEGER_AGENT_HOST` - hostname for communicating with Jaeger agent via UDP (defaults to `localhost`).
 * `JAEGER_AGENT_PORT` - port for communicating with Jaeger agent via UDP (defaults to `6831`).
 * `JAEGER_SAMPLER_TYPE` - type of sampling to use, either `probabilistic` or `const` (defaults to `probabilistic`).
 * `JAEGER_SAMPLER_PARAM` - numeric parameter used to configure the sampler type (defaults to `0.0001`).

These variables may also be set using equivalent cli flags.

By default lily uses probabilistic sampling with a rate of 0.0001. During testing it can be easier to override to remove sampling by setting
the following environment variables:

```
  JAEGER_SAMPLER_TYPE=const JAEGER_SAMPLER_PARAM=1
```

or by specifying the following flags:

```
  --jaeger-sampler-type=const jaeger-sampler-param=1
```

## Versioning and Releases

Feature branches and master are designated as **unstable** which are internal-only development builds. 

Periodically a build will be designated as **stable** and will be assigned a version number by tagging the repository
using Semantic Versioning in the following format: `vMajor.Minor.Patch`.

## Other Topics

- [Release Management](docs/release_management.md)
- [Schema/Migration Management](docs/migrations.md)

## Code of Conduct

Sentinel Visor follows the [Filecoin Project Code of Conduct](https://github.com/filecoin-project/community/blob/master/CODE_OF_CONDUCT.md). Before contributing, please acquaint yourself with our social courtesies and expectations.


## Contributing

Welcoming [new issues](https://github.com/filecoin-project/lily/issues/new) and [pull requests](https://github.com/filecoin-project/lily/pulls).


## License

The Filecoin Project and Sentinel Visor is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](https://github.com/filecoin-project/lily/blob/master/LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](https://github.com/filecoin-project/lily/blob/master/LICENSE-MIT) or http://opensource.org/licenses/MIT)
