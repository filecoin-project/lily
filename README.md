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

Install TimescaleDB v1.7.4:

In a separate shell, use docker-compose to start the appropriate version of Postgres with TimescaleDB. (Note: Visor requires TimescaleDB v1.7.x and will not work with v2.0.)

```sh
docker-compose up --build timescaledb
```

### Running tests

To quickly run tests, you can provide the `VISOR_TEST_DB` envvar and execute `make test` like so:

`VISOR_TEST_DB="postgres://postgres:password@localhost:5432/postgres?sslmode=disable" make test`

For more, manual test running, you could also prepare your environment in the following way:

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

The release commit should be tagged with an annotated and signed tag:

    git tag -asm vx.x.x vx.x.x
    git push --tags

A non-prescriptive example of the release process might look like the following:

```sh
git checkout master
git pull                                # checkout/pull latest master
git checkout -b vx.x.x-release          # create release branch
vi CHANGELOG.md                         # update CHANGELOG.md
make visor                              # validate build
go mod tidy                             # ensure tidy go.mod for release
git add CHANGELOG.md go.mod go.sum
git commit -m "chore(docs): Update CHANGELOG for vx.x.x-rc1"
                                        # commit CHANGELOG/go.mod updates
git tag -asm vx.x.x-rc1 vx.x.x-rc1      # create signed/annotated tag
git push --tags origin vx.x.x-release
                                        # push release branch and tags

# release validation

# optional hotfix flow
git commit -m "fix: Hotfix desc"        # optional hotfixes applied to release branch
vi CHANGELOG.md                         # update CHANGELOG.md
make visor                              # validate build
go mod tidy                             # ensure tidy go.mod for release
git add CHANGELOG.md go.mod go.sum
git commit -m "chore(docs): Update CHANGELOG for vx.x.x-rc2"
git tag -asm vx.x.x-rc2 vx.x.x-rc2
git push --tags origin vx.x.x-release   # push hotfix and new release candidate tag

# release acceptance

vi CHANGELOG.md
git add CHANGELOG.md
git commit -m "chore(docs): Update CHANGELOG for vx.x.x"
                                        # update/add/commit CHANGELOG.md
git tag -asm vx.x.x vx.x.x
git push --tags origin vx.x.x-release   # tag and push final release

git merge master                        # resolve upstream changes within release branch
git push origin vx.x.x-release          # push merge resolution

git checkout master
git merge vx.x.x-release
git push origin master                  # clean merge commit of release branch into master and push
```

NOTE: `sentinel-visor` pull requests prefer to be squash-merged into `master`, however considering this workflow tags release candidate within the release branch which we want to easily resolve in the repository's history, it is preferred to not squash and instead merge the release branch into `master`.


#### Maintaining CHANGELOG.md

The format is a variant of [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) combined with categories from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/). This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). The [github.com/git-chglog](https://github.com/git-chglog/git-chglog) utility assists us with maintaing CHANGELOG.md.

The sections within each release have a preferred order which prioritizes by largest-user-impact-first: `Feat > Refactor > Fix > {Area-specific or Custom Sections} > Chore`

Here is an example workflow of how CHANGELOG.md might be updated.

```sh
# checkout master and pull latest changes
git checkout master
git pull

# output the CHANGELOG content for the next release (assuming next release is v0.5.0-rc1)
go run github.com/git-chglog/git-chglog/cmd/git-chglog -o CHANGELOG_updates.md --next-tag v0.5.0-rc1

# reconcile CHANGELOG_updates.md into CHANGELOG.md applying the preferred section order
vi CHANGELOG*.md

# commit changes
rm CHANGELOG_updates.md
git add CHANGELOG.md
git commit -m 'chore(docs): Update CHANGELOG for v0.5.0-rc1'
```

Here is an [example of how the diff might look](https://github.com/filecoin-project/sentinel-visor/pull/326/commits/9536df9e39991a3b78013d1d1b36fef94562556d).

## Code of Conduct

Sentinel Visor follows the [Filecoin Project Code of Conduct](https://github.com/filecoin-project/community/blob/master/CODE_OF_CONDUCT.md). Before contributing, please acquaint yourself with our social courtesies and expectations.


## Contributing

Welcoming [new issues](https://github.com/filecoin-project/sentinel-visor/issues/new) and [pull requests](https://github.com/filecoin-project/sentinel-visor/pulls).


## License

The Filecoin Project and Sentinel Visor is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](https://github.com/filecoin-project/sentinel-visor/blob/master/LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](https://github.com/filecoin-project/sentinel-visor/blob/master/LICENSE-MIT) or http://opensource.org/licenses/MIT)
