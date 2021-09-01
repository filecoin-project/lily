# Lily Operator's Guide

## A note about names: Lily vs Visor

[The Sentinel Visor github repository](https://github.com/filecoin-project/sentinel-visor) has been renamed to "Lily". This guide refers to Visor by its new name, Lily, in all cases. This change takes effect as of release `v0.8`. Users should change instances and fragments of `lily` to `visor` for versions before `v0.8` (though we do not recommend using old versions).

For example:

- The command `visor daemon` becomes `lily daemon`.
- Environment variables `VISOR_API` becomes `LILY_API`.

## Using the examples

Console commands are expected to be executed within a local copy of the github repository unless specified otherwise. You can create a local copy of the github repository with `$ git clone https://github.com/filecoin-project/lily`.

# Context

[Lily](https://github.com/filecoin-project/lily), a node designed specifically for indexing the Filecoin blockchain, wraps [Lotus's](https://github.com/filecoin-project/lotus)* code up with additional instrumentation and extraction to get data into easier database formats for later query and analysis.

**Note: While Lily contains most/all of the capabilities of Lotus, one should note that Lily is not intended to be a replacement for Lotus. Features and performance in Lily will prioritize its primary purpose of scraping and indexing which may cause aspects of Lotus's normal behavior to be suboptimal for other use cases.*

## Common Usage Patterns

There are multiple ways to get Lily running. The fastest way to get Lily running on your local machine is to either build the binary locally or use a [pre-built Docker image](https://hub.docker.com/repository/docker/filecoin/lily).

### Minimal resources running w all Tasks

Minimum required resources depend largely on how you intend to operate Lily. We attempt to characterize the heaviest load scenario to illustrate some of the performance concerns and limitations and will leave paring back scope as an exercise to the user and their application.

#### tl;dr

The Sentinel team operates Lily on `r5.8xlarge` AWS instances and have found this sizing to accommodate the majority of the workload we ask of Lily. This instance comes with:

- 256GiB RAM
- 32vCPU running on 3.1Ghz Intel Xeon (Skylake-SP or Cascade Lake)
- 10Gbps network access
- 3Tb EBS volume (w 6,800Mbps EBS transfer rate)

A typical deployment of Lily will configure two server instances w the following four Jobs:

*Instance #1*

> `lily watch --storage=db --confidence=100 --window=30s \ --tasks="blocks,messages,chaineconomics,actorstatesraw,actorstatespower,actorstatesreward,actorstatesmultisig,msapprovals"`

> `lily watch --storage=db --confidence=100 --window=30s --tasks=actorstatesinit`

> `lily watch --storage=db --confidence=100 --window=60s --tasks=actorstatesmarket`

*Instance #2*

> `lily watch --storage=db --confidence=100 --window=60s --task=actorstatesminer`

*Important Notes*

- High confidence values will protect you from indexing data on the wrong branch during a reorg. (See [args: `lily [watch|walk] --confidence` (TODO: link needed)](#LINKNEEDED) for details.)
- Large timeout windows allow a Task which may occasionally run longer to complete instead of being terminated. (See [args: `lily [watch|walk] --window`(TODO: link needed)](#LINKNEEDED) for details.)
- The `actorstatesminer` Task produces the most intensive workload of all the Tasks. It is recommended to isolate that Task on its own Job and preferably on its own machine. The miner state of each two tipsets is loaded into memory for diffing and it is the largest of all the states.
- When multiple Tasks are assigned to the same Job, the Job will not continue to the next tipset until all Tasks are completed or skipped.
- Lily Tasks are typically memory-bound, then disk-bound before they are CPU-bound. Disk IO and CPU usage are not as highly demanded as memory. Memory-optimized hardware should be prioritized for Lily deployments.

A quick overview of Lily's operation is also available by executing `lily help overview` in the console.

### Building Lily Locally

TODO

#### Building Lily for testnet

TODO

### Running Dependencies w Docker

Once your local Lily build is ready, you can get supporting services up and running quickly using pre-built Docker containers with `docker-compose`.

*(Note: `docker`, `docker-compose`, and `make` are dependencies. See [Docker Installation](https://docs.docker.com/engine/install/) instructions and [GNU Make](https://www.gnu.org/software/make/) for installing make on your system.)*

Included in the [`docker-compose.yml`](https://github.com/filecoin-project/lily/tree/master/docker-compose.yml) are all the complimentary services that Lily might require for local debugging and development. These services come preconfigured to work with Lily's default ports.

- TimescaleDB
- Prometheus
- Grafana
- Jaeger Tracing (all-in-one)

Once these services are started, you can build and initialize Lily. A shortcut is provided within the Lily github repository to manage these services:

Examples:

# create and start dependent services
`$ docker-compose up -d`

# stop and destroy dependent services
`$ docker-compose down`

# stop (but not destroy) dependent services
`$ docker-compose stop`


### Deploy to Kubernetes (w Helm)

[Helm charts for Lily are available](https://github.com/filecoin-project/helm-charts/) for ease of deployment. The following steps will use helm to deploy Lily to an existing Kubernetes cluster.

0. [Install Helm](https://helm.sh/docs/intro/install/) on your computer. Have or [create a Kubenetes cluster](https://kubernetes.io/docs/tutorials/kubernetes-basics/create-cluster/). And configure your cluster in your environment. (See [Configuring your environment for Kubernetes (TODO: link needed)](#LINKNEEDED) for more information.)

1. Add Filecoin helm charts repo.

```
$ helm repo add filecoin https://filecoin-project.github.io/helm-charts
```

1. Copy default values YAML out of the Helm chart into a new `values.yaml` file to use locally for your deployment and adjust the values to meet the needs of your deployment. (Recommendations are provided inline. Details for each key are included.)

``` 
$ mkdir ./production-lily
$ cd production-lily
$ helm show values filecoin/lily (TODO: Doublecheck that this is updated.) > custom-values.yaml
$ vi ./custom-values.yaml
``` 

2. Configure your environment for the kubernetes cluster you intend to interact with.

```
# should match the name of the cluster as configured in `~/.kube/config`
$ export KUBE_CONTEXT=<kuberenetes_context>
# should be the kubernetes namespace you intend to deploy into
$ export KUBE_NAMESPACE=<kuberenetes_namespace>
```

_NOTE: If `--kube-context` and `--namespace` are not provided, the context and namespace are decided by kubectl config values. Here is a [helpful set of scripts](https://github.com/yankeexe/kubectx) to manage these values if you're changing them often._

3. (optional) If your deployment persists data, your database credentials must be provided in a `Secret` within the cluster. The `<deployed_secret_name>` is the key which is referred to in the `custom-values.yaml`.

```
# export the following envvars (whose names have no significance other 
# than to indicate their placement in the `kubectl` command below)
LILY_SECRET_NAME=<deployed_secret_name>
LILY_SECRET_HOST=<resolveable_hostname>
LILY_SECRET_PORT=<port>
LILY_SECRET_DATABASE=<database_name>
LILY_SECRET_USER=<username>
# (take care not to leak sensitive passwords by using them directly in the console)
LILY_SECRET_PASS=<password>

# create the secret
$ kubectl create secret generic \
    --context="$(KUBE_CONTEXT)" \
    --namespace "$(KUBE_NAMESPACE)" \
    "$(LILY_SECRET_NAME)" \
--from-literal=url="postgres://$(LILY_SECRET_USER):$(LILY_SECRET_PASS)@$(LILY_SECRET_HOST):$(LILY_SECRET_PORT)/$(LILY_SECRET_DATABASE)?sslmode=require"
```

Be sure to configure your `custom-values.yaml` with the `<deployed_secret_name>` like so:


```
...
storage:
    postgresql:
    - name: db
      secretName: <deployed_secret_name> # <- goes here
      secretKey: url
      schema: lily
      applicationName: lily
...
```

4. Deploy your release with `helm install`. (Make sure you are using the right kubernetes context for your intended cluster.) The following example uses `helm upgrade --install` which universally works for `install` and `upgrade` (change-in-place) operations.

`$ helm upgrade --install --kube-context="$(KUBE_CONTEXT)" --namespace="$(KUBE_NAMESPACE)" $(RELEASE_NAME) filecoin/lily -f ./custom-values.yaml`

With values expanded, it should look something like the following:

`$ helm upgrade --install --kube-context="arn:aws:eks:us-east-N:000000000000:cluster/custom-cluster-name" --namespace="custom-namespace" monitoring filecoin/lily -f ./custom-values.yaml`

_(NOTE: The flags `--wait` and `--timeout` can be added to make this a blocking request, instead of returning immediately after successful delivery of the install/upgrade request.)_


5. Monitor the deployment of your release.

```
# get logs of Lily container (export only one CONTAINER_NAME)
$ export CONTAINER_NAME=daemon  # default container
$ export CONTAINER_NAME=chain-import  # useful when chain-import is enabled and running long
$ kubectl --context="$(KUBE_CONTEXT)" --namespace="$(KUBE_NAMESPACE)" logs $(RELEASE_NAME)-lily-0 $(CONTAINER_NAME) --follow

# get interactive shell in Visor container
$ kubectl --context="$(KUBE_CONTEXT) --namespace="$(KUBE_NAMESPACE)" exec -it $(RELEASE_NAME)-lily-0 $(CONTAINER_NAME) -- bash
```

(TODO: Doublecheck that instance enumeration names change visor -> lily to support above example)

7. Iterate over custom `values.yaml` and deploy changes.

```
# apply changes and save
$ vi custom-values.yaml
# same helm upgrade --install command as before
$ helm upgrade --install --kube-context="$(KUBE_CONTEXT)" --namespace="$(KUBE_NAMESPACE)" $(RELEASE_NAME) filecoin/lily -f custom-values.yaml
```

#### Controlling Helm chart versions used in deployment

If you want to control the specific version of Helm chart used, a `--version` flag may be passed into `helm upgrade|install` like so (simplified for understanding):

`$ helm upgrade --install $(RELEASE_NAME) filecoin/lily --version "M.N.R"`

## Operating Lily

### Daemon First Start

Lily supports operation as a deamon process which allows stateful management of Jobs. The daemon starts without any Jobs assigned and will proceed to sync to the network and then wait until Jobs are provided. 

Typical initialization and startup for Lily will start with `lily init` to establish the datastore, params, and boilerplate config.

Once Lily is prepared, it may be started with `lily daemon` (including any custom arguments you provided with `lily init`).

The status of Lily's sync to the network can be checked with `lily sync status`, or for a blocking version `lily sync wait`.

### Job Management

#### Initialize a new Job

Once the Lily daemon process is running, you may manage its Jobs through the CLI. (Note: There is currently no way to preconfigure the daemon to start certain Jobs. If Jobs are provided while Lily is syncing to the network, the behavior will be undefined. One may gate the Job creation to wait for sync to complete with `lily sync wait && lily ...`.)

Currently, Lily accepts the following Jobs:

- `lily watch` follows along and indexes the blockchain HEAD as the network progresses.

- `lily walk` will walk the tipsets between `--to` and `--from` then stop.
- `lily gap find` walks over tables in the Postgres storage provided to the task and prepares the work needed to execute `lily gap fill`.
- `lily gap fill` uses the prepared work from `lily gap find` to execute the named `--tasks` over any detected gaps in our data.

Each Job manages a set of Tasks which are executed in parallel on each tipset. Default tasks will be assigned if not specified. (See all available Tasks at [option: Customize Tasks per Job (TODO: link needed)](#linkneeded) or execute `lily help tasks` in the console for more information.)

When one of these Jobs are executed on a running daemon, a new Job (with ID) is created. These jobs may be managed using the `lily job` CLI like so:

- `lily job list` shows all Jobs and their status
- `lily job stop` allows a running Job to be stopped 
- `lily job start` allows a stopped Job to be resumed (new Jobs are not created this way)

Examples:

```
$ lily job list
List Jobs:
[
        {
                "ID": 1,
                "Name": "customtaskname",
                "Type": "watch",
                "Error": "",
                "Tasks": [
                        "blocks",
                        "messages",
                        "chaineconomics",
                        "actorstatesraw",
                        "actorstatespower",
                        "actorstatesreward",
                        "actorstatesmultisig",
                        "msapprovals"
                ],
                "Running": true,
                "RestartOnFailure": true,
                "RestartOnCompletion": false,
                "RestartDelay": 0,
                "Params": {
                        "confidence": "100",
                        "storage": "db",
                        "window": "30s"
                },
                "StartedAt": "2021-08-27T21:56:49.045783716Z",
                "EndedAt": "0001-01-01T00:00:00Z"
        }
]
```

### Gaining Useful Insights into Lily's Performance

Understanding how Lily's progress and performance changes over time will be important for any user which may leave a daemon running to monitor the network. Lily adopts modern monitoring solutions to provide any operator with feedback on the health of Lily.

#### Processing Reports

Lily captures details about each Task completed within the configured storage in a table called `visor_processing_reports`*. This table includes the height, state_root, reporter (via [ApplicationName (TODO: link needed)](#linkneeded)), task, started/completed timestamps, status, and errors (if any). This provides task-level insight on how Lily is progressing with the provided Jobs as well as any internal errors.

_(*Note: This table name has not yet been updated to `lily_processing_reports` to minimize DB schema churn in the short-term, but will be updated once the next major schema migration is released.)_

#### Prometheus Metrics

Lily automatically exposes an HTTP endpoint which exposes internal performance metrics. The endpoint is intended to be consumed by a [Prometheus](https://prometheus.io/docs/introduction/overview/) server. (Prometheus may be locally deployed using Docker as described in [Running Dependencies w Docker (TODO: Link needed)](#linkneeded).)

Prometheus metrics are exposed by default on `http://0.0.0.0:9991/metrics` and may be bound to a custom IP/port by passing `--prometheus-port="0.0.0.0:9991"` on daemon startup with your custom values. (See [option: Custom Prometheus endpoint (TODO: link needed))[#linkneeded] or execute `lily help monitoring` in the console for more information.)

Example:

# bind to all local interfaces on port 9991
`$ lily daemon --prometheus-port=":9991"`

# bind to a specific IP on a custom port
`$ lily daemon --prometheus-port="10.0.0.1:9991"`

A description of these metrics are included inline in the reply. A sample may be captured using curl:

`$ curl 0.0.0.0:9991/metrics -o lily_prom_sample.txt`

#### Pre-built Grafana Dashboards

Grafana is a metrics visualization tool which requests, transforms, and presents the data captured within your Prometheus and storage database. [Pre-build dashboards are available (TODO: link needed)](#linkneeded) in the github repository which can be imported into your own instance of Grafana. Or you can start a copy of Grafana with the dashboards pre-loaded as described in [Running Dependencies w Docker (TODO: link needed)](#linkneeded). The panels have titles, labels, and descriptions (TODO: doublecheck and add panel descriptions to the dashboards in the repo) which describe what the visualization is for.

(TODO: Upload and add a screenshot.)

#### Logging

Lily emits logs about each module contained within the runtime. Their level of verbosity can be managed on a per-module basis. A full list of registered modules can be retrieved on the console with `$ lily log list`. All modules have defaults set to prevent verbose output. Logging levels are one of `DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL`. Logging can also be generated in colorized text (default), plain text, and JSON. (See [config: Node Logging options (TODO: link needed)](#linkneeded) for more information.)

Examples:
```
# set level for all modules via envvar
$ explore GOLOG_LOG_LEVEL="debug"

# set levels for multiple modules via envvar (can be different levels per module)
$ export GOLOG_LOG_LEVEL_NAMED="chain:debug,chainxchg:info"

# set levels for multiple modules via arg on startup
$ lily daemon --log-level-named="chain:debug,chainxchg:info"

# set levels for multiple modules via CLI (requires daemon to be running, one level per command)
$ lily log set-level --system chain --system chainxchg debug
```

#### Jaeger Tracing

Lily is capable of exposing traces during normal runtime. This behavior is disabled by default because there is a performance impact for these traces to be captured. These traces are produced using Jaeger and are compatible with [OpenCensus](https://opencensus.io/tracing/). 

Jaeger tracing can be enabled by passing the `--tracing` flag on daemon startup. There are other configuration values which have "reasonable" default values, but should be reviewed for your use case before enabling tracing. (See [option: Custom tracing configuration (TODO: link needed))[#linkneeded] or execute `lily help monitoring` in the console for more information.)

#### Capturing Profiles

Lily exposes runtime profiling endpoints during normal runtime. This behavior is always available, but waits for interaction through the exposed HTTP endpoint before capturing this data.
By default, the profiling endpoint is exposed at `http://0.0.0.0:1234/debug/pprof`. This will serve up valid HTML to be viewed through a browser client or this endpoint can be connected to using the `go pprof tool` using the appropriate endpoint for the type of profile to be captured. (See [interacting with the pprof HTTP endpoint](https://pkg.go.dev/net/http/pprof) for more information.)

Example:

# capture local heap profile and load into pprof for analysis
```
$ curl 0.0.0.0:1234/debug/pprof/heap -o heap.pprof.out
$ go tool pprof ./path/to/binary ./heap.pprof.out
```

# inspect profile interactively via `http://localhost:1234/debug/pprof` and host a web interface at `http://localhost:8000` (which opens automatically once profile is captured)
```
$ go tool pprof -http :8000 :1234/debug/pprof/heap
```

## Common Configuration Options

(TODO: I'm expecting there to be overlap w regular Lily docs here. Leaving this for last.)

The options described here are not exhaustive and focus on those most helpful to operator usage. Please refer to the Lily Documentation for full details on each option.

### Configuration Precedence

There are many ways to define certain options in Lily and precedence is defined in the following order:

0. Command line flag value from user
1. Environment variable (if specified)
2. Configuration file (if specified)
3. Default defined on the flag

### `lily init` Options

#### option: Import Snapshot

arg: `lily init --import-snapshot`
env: `$LILY_SNAPSHOT`

Import a snapshot to quickly bootstrap a new node syncing to a network. (This is especially useful for long-lived networks, such as `mainnet`.)

Protocol Labs maintains a snapshot for `mainnet` at [https://docs.filecoin.io/get-started/lotus/chain/#syncing](https://docs.filecoin.io/get-started/lotus/chain/#syncing). (*Note: A lightweight snapshot will not contain complete historical state and will fail to work for historical indexing. Please double-check the snapshot chosen for import.*)

Example:

`$ lily init --import-snapshot="https://fil-chain-snapshots-fallback.s3.amazonaws.com/mainnet/minimal_finality_stateroots_latest.car"`

#### option: Initialize Repo Path

arg: `lily init --repo`
env: `$LILY_REPO`
default: `$HOME/.lily`

Create a new repo at a custom path for Lily's local state on the filesystem (also known as the `repo`). The repo contains chain/block data, keys, configuration, and other operational data. This folder is portable and can be relocated as needed.

If a directory already exists at the location, it will be untouched by `lily init`.

Example:

`$ lily init --repo=/var/lib/lily`

#### option: Initialize Config

arg: `lily init --config`
env: `$LILY_CONFIG`

Create a new configuration template at the location specified. The file uses the TOML format. This file is portable and can be relocated as needed.

If a configuration file already exists at the location, it will be untouched by `lily init`.

Example:
`$ lily init --config=/var/lib/lily/config.toml`

### Args for `lily daemon`

#### option: Daemon w custom Repo Path

arg: `lily daemon --repo`
env: `$LILY_REPO`

Specify a custom path for Lily to read and manage its state on the filesystem (also known as the `repo`). The repo contains chain/block data, keys, configuration, and other operational data. This folder is portable and can be relocated as needed.

Example:
`$ lily daemon --repo=/var/lib/lily`

#### option: Daemon w custom Config Path

arg: `lily daemon --config`
env: `$LILY_CONFIG`

Specify a custom configuration path for the daemon to use during runtime. The file uses the TOML format. This file is portable and can be relocated as needed.

Example:
`$ lily daemon --config=/var/lib/lily/config.toml`

#### option: Daemon JSON API IP/port

arg: `lily daemon --api`
env: `$LILY_API`
config: `API.ListenAddress`
default: `/ip4/127.0.0.1/tcp/1234`

Specify a custom IP and port for the JSON RPC API to locally bind to. This string should be provided as a [multiaddr](https://multiformats.io/multiaddr/) which describes to Lily how it should expose its JSON RPC API.

Example:

(arg)
`$ lily daemon --api="/ip4/127.0.0.1/tcp/1234"`

(config)
```
[API]
  ...
  ListenAddress = "/ip4/127.0.0.1/tcp/1234"
  ...
```

### Args for `lily watch` and `lily walk`

Note: Some arguments are shared between multiple Jobs and are indicated in the heading.

#### args: `lily [walk|watch] --confidence`

Context: A network with distributed consensus may occasionally have intermittent connectivity problems which cause some nodes to have different views of the true blockchain HEAD. Eventually, the connectivity issues resolve, the nodes connect, and they reconcile their differences. The node which is found to be on the wrong branch will reorganize its local state to match the new network consensus by "unwinding" the incorrect chain of tipsets up to the point of disagreement (the "fork") and then applying the correct chain of tipsets. This can be referred to as a "reorg". The number of tipsets which are "unwound" from the incorrect chain is referred to as "reorg depth".

Lily makes use of a "confidence" FIFO cache which gives the operator confidence that the tipsets which are being processed and persisted are unlikely to be reorganized. A confidence of 100 would establish a cache which will fill with as many tipsets. Once the 101st tipset is unshifted onto the cache stack, the 1st tipset would be popped off the bottom and have the Tasks processed over it. In the event of a reorg, the most recent tipsets are shifted off the top and the correct tipsets are unshifted in their place.

Example:

`$ lily watch --confidence=100`

A visualization of the confidence cache during normal operation:

```

             *unshift*        *unshift*      *unshift*       *unshift*
                │  │            │  │            │  │            │  │
             ┌──▼──▼──┐      ┌──▼──▼──┐      ┌──▼──▼──┐      ┌──▼──▼──┐
             │        │      │  ts10  │      │  ts11  │      │  ts12  │
   ...  ---> ├────────┤ ---> ├────────┤ ---> ├────────┤ ---> ├────────┤ --->  ...
             │  ts09  │      │  ts09  │      │  ts10  │      │  ts11  │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ts08  │      │  ts08  │      │  ts09  │      │  ts10  │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ...   │      │  ...   │      │  ...   │      │  ...   │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ts02  │      │  ts02  │      │  ts03  │      │  ts04  │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ts01  │      │  ts01  │      │  ts02  │      │  ts03  │
             ├────────┤      ├────────┤      ├────────┤      ├────────┤
             │  ts00  │      │  ts00  │      │  ts01  │      │  ts02  │
             └────────┘      └────────┘      └──│──│──┘      └──│──│──┘
                                                ▼  ▼  *pop*     ▼  ▼  *pop*
                                             ┌────────┐      ┌────────┐
              (confidence=10 :: length=10)   │  ts00  │      │  ts01  │
                                             └────────┘      └────────┘
                                              (process)       (process)
```

A visualization of the confidence cache during a reorg of depth=2:

```

  *unshift*    *shift*    *shift*  *unshift*  *unshift*  *unshift*
     │  │       ▲  ▲       ▲  ▲       │  │       │  │       │  │
   ┌─▼──▼─┐   ┌─│──│─┐   ┌─│──│─┐   ┌─│──│─┐   ┌─▼──▼─┐   ┌─▼──▼─┐
   │ ts10 │   │      │   │ │  │ │   │ │  │ │   │ ts10'│   │ ts11'│
   ├──────┤   ├──────┤   ├─│──│─┤   ├─▼──▼─┤   ├──────┤   ├──────┤
   │ ts09 │   │ ts09 │   │      │   │ ts09'│   │ ts09'│   │ ts10'│
   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤
   │ ts08 │   │ ts08 │   │ ts08 │   │ ts08 │   │ ts08 │   │ ts09'│
   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤
   │ ...  │ > │ ...  │ > │ ...  │ > │ ...  │ > │ ...  │ > │ ...  │
   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤
   │ ts02 │   │ ts02 │   │ ts02 │   │ ts02 │   │ ts02 │   │ ts03 │
   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤
   │ ts01 │   │ ts01 │   │ ts01 │   │ ts01 │   │ ts01 │   │ ts02 │
   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤   ├──────┤
   │ ts00 │   │ ts00 │   │ ts00 │   │ ts00 │   │ ts00 │   │ ts01 │
   └──────┘   └──────┘   └──────┘   └──────┘   └──────┘   └─│──│─┘
                                                            ▼  ▼  *pop*
               reorg                            reorg     ┌──────┐
               occurs                          resolves   │ ts00 │
                here                             here     └──────┘
                                                          (process)
```

Note: A large confidence protects you from large reorgs but causes a longer delay between startup and processing Tasks on a fully synced Lily node.

Note: A small (or zero) confidence will allow tipsets which are reorged to be persisted despite only appearing on-chain for a brief time. This may be useful when attempting to analyze differences in state during reorgs.

#### option: Task Timeout Window

arg: `lily [walk|watch] --window`
default: `30s`

Configure a custom duration in which Lily does as much task processing as possible, any task(s) not completed within the window will be marked as incomplete. This means some epochs may not contain all task data. (And are candidates for later re-processing via gap find & fill.) Each `walk|watch` Job manages its own `window` value. This value is provided as [a parseable Golang duration](https://pkg.go.dev/time#ParseDuration).

Example:

```
# passed as arg
$ lily watch --window=60s
```

#### option: Persistence Config

config: `Storage.Postgresql|File.[Name]` (object)

Lily can deliver scraped data to multiple PostgreSQL and File destinations on a per-Task basis. Each destination should be enumerated with a unique `[Name]` which will be used as an argument when starting a Task.

*Note: Duplicate names among both PostgreSQL and File destinations will have undefined behavior.*

Example:
```
[Storage]
  [Storage.Postgresql]
    [Storage.Postgresql.Name1]
      URL = "postgres://postgres:password@localhost:5432/primarydatabase"
      ApplicationName = "lily"
      SchemaName = "public"
      PoolSize = 20
      AllowUpsert = false
    [Storage.Postgresql.Name2]
      URL = "postgres://postgres:password@localhost:5432/anotherdatabase"
      ApplicationName = "lily"
      SchemaName = "public"
      PoolSize = 10
      AllowUpsert = false
  [Storage.File]
    [Storage.File.CSV]
      Format = "CSV"
      Path = "/tmp"
      OmitHeader = false
      FilePattern = "{table}.csv"
    [Storage.File.CSV2]
      Format = "CSV"
      Path = "/output"
      OmitHeader = false
      FilePattern = "{table}.csv"
```

#### option: Postgres Connection URL

config: `Storage.Postgresql.[Name].URL` (string)

A PostgreSQL connection string for this destination which aheres to [this spec](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING).

#### option: Postgres Connection Environment Variable

config: `Storage.Postgresql.[Name].URLEnv` (string)

The name of the environment variable which contains a valid PostgreSQL connection string. If this value is present, it will be preferred as the connection string over the value provided in `Storage.Postgresql.[Name].URL`.

#### option: Postgres Connection Application Name

config: `Storage.Postgresql.[Name].ApplicationName` (string)

This is the name provided within progress reports delivered to the PostgreSQL destination. Providing a unique name here will allow multiple jobs to feed the same database while still able to identify to which instance the report belongs.

## Troubleshooting

### Configuring your environment for Kubernetes

Also sometimes appears as the following deployment error: "Error: Kubernetes cluster unreachable: context "<name>" does not exist".

Background: When deploying to Kubernetes (using `helm` or `kubectl`) a `kube-context` is required to indicate which cluster the current operation should be applied to. If `kube-context` is not provided, the `default` context is assumed and generally works fine. But if a custom context is provided and has not been configured in your local environment yet, you may get the error `context "<name>" does not exist`.

To configure your local environment for AWS EKS cluster:

Make sure you have an AWS account setup with access keys that have privileges to access AWS EKS.
Install the following:
[AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
[Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
[Helm](https://helm.sh/docs/intro/install/)
In `~/.bashrc` set AWS access keys:
`$ export AWS_ACCESS_KEY_ID=”<key>”`
`$ export AWS_SECRET_ACCESS_KEY="<secret>”`
`$ export AWS_REGION=”<awsregion>”`
Reload `~/.bashrc`:
`$ source ~/.bashrc`
Follow [this guide](https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html) to setup kubectl config. Here's an overview of the steps:
Verify version 1.16.156 or later; python 2.7.9 or later:
`$ aws --version`
Setup configuration to work with AWS EKS:
`$ aws eks --region $AWS_REGION update-kubeconfig --name <aws-eks-custom-name>`
Test your kubectl configuration:
`$ kubectl get nodes`
Test helm can show current releases (may be an empty list, just looking for no error):
`$ helm ls`

To configure your local environment for Kubernetes that isn't on AWS, refer to the [cluster setup documentation for local development](https://kubernetes.io/docs/tasks/tools/).

### Kubernetes Deployment Issues

#### deployment error: "N node(s) had volume node affinity conflict"

Background: Sentinel Visor has Persistent Volume Claims (PVC) which are lazily assigned as they bound to the pod they are requested from. When a release has been destroyed without cleaning these lazily created PVC or an upgrade causes a pod to be assigned to a new node (different from what it was previously scheduled) the existing PVC can cause the deployment to become stuck.

To resolve:

1. Describe the pod which is having the schedule confict to identify the PVC name it is bound to. _(Note: Assuming your release name is `analysis`.)_

```
kubectl describe pod <releasename>-visor-0
```

Example:
```
$ kubectl describe pod analysis-visor-0
...
...
Volumes:
  datastore-volume:
    Type:       PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)
    ClaimName:  datastore-volume-analysis-visor-0
    ReadOnly:   false
...
...
```

2. Delete the PVC.

```
kubectl delete pvc datastore-volume-<releasename>-visor-0
```

Example:
```
$ kubectl delete pvc datastore-volume-analysis-visor-0
persistentvolumeclaim "datastore-volume-analysis-visor-1" deleted
```

3. Restart the pod.

```
kubectl delete pod <releasename>-visor-0
```

Example:
```
$ kubectl delete pod analysis-visor-0
pod "analysis-visor-1" deleted
```

#### deployment error: Multi-Attach error for volume "XXX"

Background: Deployment is in progress and the new pod attempts to start up but blocks with the error "Multi-Attach error for volume "pvc-a41e35bd-d1dc-11e8-9b2b-fa163ef89d28" Volume is already exclusively attached to one node and can't be attached to another." This is likely a timing issue where termination of a pod on one node has not allowed the volume to be released before the scheduling and spin-up of the replacement pod on a different node. The quickest fix is to delete the pod and allow the StatefulSet to restore the pod and be able to bind the volume to the new node.

To resolve:

1. Delete the pod and wait for the scheduler to start it again. Generally, it only takes a little time for the volume to release.

```
$ kubectl delete pod <releasename>-visor-<N>
```

Example:
```
$ kubectl delete pod analysis-visor-1
pod "analysis-visor-1" deleted
```

2. Observe the pod being rescheduled and the volume properly attaching to the new node.

