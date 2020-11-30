# Sentinel Visor
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/filecoin-project/sentinel-visor) [![docker build status](https://img.shields.io/docker/cloud/build/filecoin/sentinel-visor?style=flat-square)](https://hub.docker.com/repository/docker/filecoin/sentinel-visor) [![CI build](https://img.shields.io/circleci/build/gh/filecoin-project/sentinel-visor?label=ci%20build&style=flat-square)](https://app.circleci.com/pipelines/github/filecoin-project/sentinel-visor)

A component of [**Sentinel**](https://github.com/filecoin-project/sentinel), a collection of services which monitor the health and function of the Filecoin network. 

A **Visor** process collects _permanent_ Filecoin chain meterics from a [**Lotus**](https://github.com/filecoin-project/lotus/) daemon, and writes them to a [**TimescaleDB**](https://github.com/timescale/timescaledb) time-series and relational datastore.


## Getting Started

Clone the repo and build the dependencies:

```console
$ git clone git@github.com:filecoin-project/sentinel-visor.git
$ cd sentinel-visor
$ make deps
```

Build the `sentinel-visor` binary to the root of the project directory:

```console
$ make build
```

Install PostgreSQL:

```sh
brew install postgresql@12
```

Install TimescaleDB:

```sh
brew tap timescale/tap
brew install timescaledb
```

You may need to run the following to get timescale to compile:

```sh
ln -s /usr/local/Cellar/postgresql@12/12.5/include/postgresql/server /usr/local/Cellar/postgresql@12/12.5/include/server
```

Activate TimescaleDB by adding `shared_preload_libraries = 'timescaledb'` to `postgresql.conf`. Restart the postgres server if it's running.

### Running tests

Create a new DB in postgres for testing:

```sql
CREATE DATABASE visor_test;
```

Migrate the database to the latest schema:

```sh
visor --db "postgres://username@localhost/visor_test?sslmode=disable" migrate --latest
```

Run the tests:

```sh
VISOR_TEST_DB="postgres://username@localhost/visor_test?sslmode=disable" go test ./...
```

### Usage

```
  sentinel-visor [<flags>] <command>

  Use 'sentinel-visor help <command>' to learn more about each command.
```

Use the following env vars to configure the lotus node that visor reads from, and the database that it writes to:

- `LOTUS_PATH` - path to the lotus data dir. _default: `~/.lotus`_
- `LOTUS_DB` - database connection . _default: `postgres://postgres:password@localhost:5432/postgres?sslmode=disable`_

### Configuring Tracing

The global flag `--tracing=<bool>` turns tracing on or off. It is on by default.

Tracing expects a Jaeger server to be available. Configure the Jaeger settings using the following subset of the standard Jaeger [environment variables](https://github.com/jaegertracing/jaeger-client-go#environment-variables):

 * `JAEGER_SERVICE_NAME` - name of the service (defaults to `sentinel-visor`).
 * `JAEGER_AGENT_HOST` - hostname for communicating with Jaeger agent via UDP (defaults to `localhost`).
 * `JAEGER_AGENT_PORT` - port for communicating with Jaeger agent via UDP (defaults to `6831`).
 * `JAEGER_SAMPLER_TYPE` - type of sampling to use, either `probabilistic` or `const` (defaults to `probabilistic`).
 * `JAEGER_SAMPLER_PARAM` - numeric parameter used to configure the sampler type (defaults to `0.0001`).

These variables may also be set using equivalent cli flags.

By default visor uses probabilistic sampling with a rate of 0.0001. During testing it can be easier to override to remove sampling by setting
the following environment variables:

```
  JAEGER_SAMPLER_TYPE=const JAEGER_SAMPLER_PARAM=1
```

or by specifying the following flags:

```
  --jaeger-sampler-type=const jaeger-sampler-param=1
```

## Deployment

### Schema Migrations

The database schema is versioned and every change requires a migration script to be executed. See [storage/migrations/README.md](storage/migrations/README.md) for more information.

### Checking current schema version

The visor `migrate` subcommand compares the **database schema version** to the **latest schema version** and reports any differences.
It also verifies that the **database schema** matches the requirements of the models used by visor. It is safe to run and will not alter the database.

Visor also verifies that the schema is compatible when the index or process subcommands are executed.

### Migrating schema to latest version

To migrate a database schema to the latest version, run:

    visor migrate --latest

Visor will only migrate a schema if it determines that it has exclusive access to the database. 

Visor can also be configured to automatically migrate the database when indexing or processing by passing the `--allow-schema-migration` flag.

### Reverting a schema migration

To revert to an earlier version, run:

    visor migrate --to <version>

**WARNING: reverting a migration is very likely to lose data in tables and columns that are not present in the earlier version**


## Versioning and Releases

Feature branches and master are designated as **unstable** which are internal-only development builds. 

Periodically a build will be designated as **stable** and will be assigned a version number by tagging the repository
using Semantic Versioning in the following format: `vMajor.Minor.Patch`.
 
### Release Process

Between releases we keep track of notable changes in CHANGELOG.md.

When we want to make a release we should update CHANGELOG.md to contain the release notes for the planned release in a section for
the proposed release number. This update is the commit that will be tagged with as the actual release which ensures that each release
contains a copy of it's own release notes. 

We should also copy the release notes to the Github releases page, but CHANGELOG.md is the primary place to keep the release notes. 

The release commit should be tagged with a signed tag:

    git tag -s vx.x.x
    git push --tags



## Code of Conduct

Sentinel Visor follows the [Filecoin Project Code of Conduct](https://github.com/filecoin-project/community/blob/master/CODE_OF_CONDUCT.md). Before contributing, please acquaint yourself with our social courtesies and expectations.


## Contributing

Welcoming [new issues](https://github.com/filecoin-project/sentinel-visor/issues/new) and [pull requests](https://github.com/filecoin-project/sentinel-visor/pulls).


## License

The Filecoin Project and Sentinel Visor is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](https://github.com/filecoin-project/sentinel-visor/blob/master/LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](https://github.com/filecoin-project/sentinel-visor/blob/master/LICENSE-MIT) or http://opensource.org/licenses/MIT)
