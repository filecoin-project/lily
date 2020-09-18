# Sentinel Visor
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/filecoin-project/sentinel-visor)

A component of [**Sentinel**](https://github.com/filecoin-project/sentinel), a collection of services which monitor the health and function of the Filecoin network. 

A **Visor** process collects _permanent_ Filecoin chain meterics from a [**Lotus**](https://github.com/filecoin-project/lotus/) daemon, and writes them to a [**TimescaleDB**](https://github.com/timescale/timescaledb) time-series and relational datastore.


## Getting Started

### Usage

```
  sentinel-visor [<flags>] <command>

  Use 'sentinel-visor help <command>' to learn more about each command.
```

### Configuring Tracing

The global flag `--tracing=<bool>` turns tracing on or off. It is on by default.

Tracing expects a Jaeger server to be available. Configure the Jaeger settings using [environment variables](https://github.com/jaegertracing/jaeger-client-go#environment-variables). The most important are:

 * `JAEGER_SERVICE_NAME` - name of the service (defaults to `sentinel-visor`).
 * `JAEGER_AGENT_HOST` - hostname for communicating with Jaeger agent via UDP (defaults to `localhost`).
 * `JAEGER_AGENT_PORT` - port for communicating with Jaeger agent via UDP (defaults to `6831`).

By default visor queries the Jaeger agent to determine what level of sampling is required (known as the `remote` sampling type). During testing it can be easier to override to remove sampling by setting
the following environment variables:

```
  JAEGER_SAMPLER_TYPE=const JAEGER_SAMPLER_PARAM=1
```

## Code of Conduct

Sentinel Visor follows the [Filecoin Project Code of Conduct](https://github.com/filecoin-project/community/blob/master/CODE_OF_CONDUCT.md). Before contributing, please acquaint yourself with our social courtesies and expectations.


## Contributing

Welcoming [new issues](https://github.com/filecoin-project/sentinel-visor/issues/new) and [pull requests](https://github.com/filecoin-project/sentinel-visor/pulls).


## License

The Filecoin Project and Sentinel Visor is dual-licensed under Apache 2.0 and MIT terms:

- Apache License, Version 2.0, ([LICENSE-APACHE](https://github.com/filecoin-project/sentinel-visor/blob/master/LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](https://github.com/filecoin-project/sentinel-visor/blob/master/LICENSE-MIT) or http://opensource.org/licenses/MIT)
