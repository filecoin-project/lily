# Sentinel Visor
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/filecoin-project/sentinel-visor)

A component of [**Sentinel**](https://github.com/filecoin-project/sentinel), a collection of services which monitor the health and function of the Filecoin network. 

A **Visor** process collects _permanent_ Filecoin chain meterics from a [**Lotus**](https://github.com/filecoin-project/lotus/) daemon, and writes them to a [**TimescaleDB**](https://github.com/timescale/timescaledb) time-series and relational datastore.


## Getting Started

Clone the repo and run `make build`:

```console
$ git clone git@github.com:filecoin-project/sentinel-visor.git
$ cd sentinel-visor
$ make build
```

This will fetch the git modules, build the filecoin-ffi, and build a `sentinel-visor` binary to the root of the project directory.

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

## Code of Conduct

Sentinel Visor follows the [Filecoin Project Code of Conduct](https://github.com/filecoin-project/community/blob/master/CODE_OF_CONDUCT.md). Before contributing, please acquaint yourself with our social courtesies and expectations.


## Contributing

Welcoming [new issues](https://github.com/filecoin-project/sentinel-visor/issues/new) and [pull requests](https://github.com/filecoin-project/sentinel-visor/pulls).


## License

The Filecoin Project and Sentinel Visor is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](https://github.com/filecoin-project/sentinel-visor/blob/master/LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](https://github.com/filecoin-project/sentinel-visor/blob/master/LICENSE-MIT) or http://opensource.org/licenses/MIT)
