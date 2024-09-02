# Lily
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/filecoin-project/lily) [![docker build status](https://img.shields.io/docker/cloud/build/filecoin/lily?style=flat-square)](https://hub.docker.com/repository/docker/filecoin/lily) [![CI build](https://img.shields.io/circleci/build/gh/filecoin-project/lily?label=ci%20build&style=flat-square)](https://app.circleci.com/pipelines/github/filecoin-project/lily)

A component of [**Sentinel**](https://github.com/filecoin-project/sentinel), a collection of services which monitor the health and function of the Filecoin network. 

Lily is a instrumentalized instance of a [**Lotus**](https://github.com/filecoin-project/lotus/) node that collects _permanent_ Filecoin chain metrics and writes them to a [**TimescaleDB**](https://github.com/timescale/timescaledb) time-series and relational datastore or to CSV files.

## User documentation

Lily documentation, including with [build](https://lily.starboard.ventures/software/lily/setup/), [operation instructions](https://lily.starboard.ventures/software/lily/operation/), [data models](https://lily.starboard.ventures/data/models/) and [access to data dumps](https://lily.starboard.ventures/data/dumps/) is available at https://lily.starboard.ventures/.

## Running tests

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


## Metrics, tracing and debugging

See https://lily.starboard.ventures/software/lily/operation/#metrics--debugging.

## Versioning and Releases

Feature branches and master are designated as **unstable** which are internal-only development builds. 

Periodically a build will be designated as **stable** and will be assigned a version number by tagging the repository
using Semantic Versioning in the following format: `vMajor.Minor.Patch`.

## Other Topics

- [Release Management](docs/release_management.md)
- [Schema/Migration Management](docs/migrations.md)

## Code of Conduct

Lily follows the [Filecoin Project Code of Conduct](https://github.com/filecoin-project/community/blob/master/CODE_OF_CONDUCT.md). Before contributing, please acquaint yourself with our social courtesies and expectations.


## Contributing

Welcoming [new issues](https://github.com/filecoin-project/lily/issues/new) and [pull requests](https://github.com/filecoin-project/lily/pulls).


## License

The Filecoin Project and Lily are dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](https://github.com/filecoin-project/lily/blob/master/LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](https://github.com/filecoin-project/lily/blob/master/LICENSE-MIT) or http://opensource.org/licenses/MIT)
