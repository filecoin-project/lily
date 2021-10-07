# Lily

A Filecoin daemon designed to capture on-chain state from the Filecoin Network.

## Overview

At its core, Lily functions similarly to a Lotus node. It was named such as the Lily and Lotus flowers are often confused with each other. But, much like the flowers, these implementations are different.

- How does Lily work?
  - Lily is an application designed to capture on-chain-state from the Filecoin network. It runs as a daemon process wrapping a Lotus node, maintaining its own local blockstore, and synchronizing it with the Filecoin network. Lily uses its blockstore to efficiently index the Filecoin blockchain, and is capable of running different types of jobs: watch, walk, gap-find, and gap-fill. Each Job can run a variety of tasks that extract different parts of the chain. The data these tasks extract may be indexed in different storage back-ends: TimescaleDB and CSV files.
- Who should use Lily?
  - Users who wish to extract or inspect pieces of the Filecoin blockchain will find value in running a Lily node. The following examples are some of the uses cases Lily can fulfill:
    - Indexing Blocks, Messages, and Receipts
    - Deriving a view of on-chain Consensus
    - Extracting miner sectors life-cycle events: Pre-Commit Add, Sector Add, Extend, Fault, Recovering, Recovered, Expiration, and Termination
    - Collecting Internal message sends not appearing on chain. For example, Multisig actor sends, Cron Event Ticks, Block and Reward Messages.
- But I don't want to run another daemon! Can I just get a copy the data?
  - Fear not, an updated-daily archive of the information extracted by Lily is always available:
    - https://lily-data.s3.us-east-2.amazonaws.com/ // TODO use an IPNS link

## Build & Install

### Dependencies

First you'll need to install Lily's dependencies, usually provided by your distribution.

- Ubuntu

  - ```bash
    sudo apt install mesa-opencl-icd ocl-icd-opencl-dev gcc git bzr jq pkg-config curl clang build-essential hwloc libhwloc-dev wget -y && sudo apt upgrade -y
    ```

- Arch

  - ```bash
    sudo pacman -Syu opencl-icd-loader gcc git bzr jq pkg-config opencl-icd-loader opencl-headers opencl-nvidia hwloc
    ```

- Fedora

  - ```bash
    sudo dnf -y install gcc make git bzr jq pkgconfig mesa-libOpenCL mesa-libOpenCL-devel opencl-headers ocl-icd ocl-icd-devel clang llvm wget hwloc libhwloc-dev
    ```

- Amazon Linux 2

  - ```bash
    sudo yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm; sudo yum install -y git gcc bzr jq pkgconfig clang llvm mesa-libGL-devel opencl-headers ocl-icd ocl-icd-devel hwloc-devel
    ```

- [Rustup](https://rustup.rs/)

  - ```bash
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
    ```

- [Go](https://golang.org/dl/) (version 1.16.4 or higher)

  -  // [TODO go 1.17.* fails to build Lily](https://github.com/filecoin-project/lily/issues/680)

  - ```bash
    wget -c https://golang.org/dl/go1.16.4.linux-amd64.tar.gz -O - | sudo tar -xz -C /usr/local 
    ```

### Build & Install

If you are located in China be sure to check out [these helpful tips](https://docs.filecoin.io/get-started/lotus/tips-running-in-china/)

1. Clone the Repo

   - ```bash
     git clone https://github.com/filecoin-project/lily.git
     cd ./lily
     ```

2. Checkout a [release](https://github.com/filecoin-project/lily/releases)

   - ```bash
     git checkout <release_tag>
     ```

3. Build Lily

   - For Mainnet

     - ```bash
       make clean all
       ```

   - For Calibration-Net

     - ```bash
       make clean calibnet
       ```

   - // TODO other networks, once we are sure they can be supported

4. Install Lily (optional) 

   - ```bash
     cp ./lily /usr/local/bin/lily
     ```

## Environment

Lily can be configured to index the data it extracts to Postgres with TimescaleDB, provide metrics on its performance via Prometheus & Grafana, and emit Tracing data to Jaeger. 

It is assumed that you have installed `docker` and `docker-compose` on your machine, if you need a guide try [this](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-compose-on-ubuntu-20-04). 

Lastly, these steps are optional, if you just want to run Lily without extracting data skip this section, or already have the required infrastructure in place, skip this section.

### docker services

The Lily repo comes with a `docker-compose.yaml` file capable of building the TimescaleDB, Prometheus, Grafana, and Jaeger services.

- Bring up services

  - ```bash
    make dockerup
    ```

- Take down services

  - ```bash
    make dockerdown
    ```

- Postgres (w/ TimescaleDB)

  - `localhost:5432`
  - Username: `postgres`
  - Password `passowrd`

- Grafana UI

  - `localhost:3000`
  - Username: `admin`
  - Password: `admin`

- Prometheus UI

  - `localhost:9090`

- Jaeger UI
  - `localhost:16686`

## Usage

This section aims to cover the basics on how to initialize and run Lily.

### Initialize

Initialize the Lily repo and an example configuration file. 

```bash
lily init --repo=$HOME/lily --config=$HOME/config.toml
```

Or initialize Lily from a chain snapshot:

```bash
lily init --repo=$HOME/lily --config=$HOME/config.toml --import-snapshot=https://fil-chain-snapshots-fallback.s3.amazonaws.com/mainnet/complete_chain_with_finality_stateroots_latest.car
```

 ### Configure

Modify the `[Storage]` section of the configuration file to meet your needs. For a local deployment (using the aforementioned docker services) it should look similar to:

```toml
[Storage]
  [Storage.Postgresql]
    [Storage.Postgresql.LocalDatabase]
      URL = "postgres://postgres:password@localhost:5432/postgres?sslmode=disable"
      ApplicationName = "lily_demo"
      SchemaName = "public"
      PoolSize = 20
      AllowUpsert = false
```

Ensure the database Lily is configured to use is up to date with the latest schema

```bash
lily migrate --db="postgres://postgres:password@localhost:5432/postgres?sslmode=disable" --latest
```

```bash
INFO    lily/commands   commands/setup.go:126   Lily version:demo
INFO    lily/storage    storage/migrate.go:187  current database schema is version 0.0
INFO    lily/storage    storage/migrate.go:223  creating base schema for major version 1
INFO    lily/storage    storage/migrate.go:260  running schema migration from version 1.0 to version 1.3
INFO    lily/storage    storage/migrate.go:268  current database schema is now version 1.3
```



Optionally, verify the database version

```bash
lily migrate --db="postgres://postgres:password@localhost:5432/postgres?sslmode=disable"
```

```bash
INFO    lily/commands   commands/setup.go:126   Lily version:demo
INFO    lily/commands   commands/migrate.go:114 current database schema is version 1.3, latest is 1.3
INFO    lily/commands   commands/migrate.go:120 database schema is supported by this version of lily
```

### Starting the Daemon

Start the Lily daemon

```bash
lily daemon --repo=$HOME/lily/ --config=$HOME/config.toml
```

```bash
INFO    lily/commands   commands/setup.go:132   Lily version:v0.8.0+7-gfd39cd1-dirty
INFO    lily/commands   commands/daemon.go:172  lily repo: /home/frrist/lily/
INFO    lily/commands   commands/daemon.go:187  lily config: /home/frrist/config.toml
INFO    lily/config     config/config.go:145    reading config from /home/frrist/config.toml
INFO    lily/schedule   schedule/scheduler.go:175       Starting Scheduler
DEBUG   lily/storage    storage/catalog.go:28   registering storage     {"name": "LocalDatabase", "type": "postgresql"}

```

#### Checking Sync Status

There are two ways to check your Lily daemon's chain synying progress.

##### Sync status

Use `sync status` to output the current state of your local chain:

```bash
lily sync status
```

```bash
sync status:
worker 0:
        Base:   [bafy2bzacebvqom65iqcyzqp6b63svkrseqrskzaezn6trzdx3khwkc27wo6ou bafy2bzacecjy5clzvixqjg6feh2wfpixlqb3yq2lz447v5m35rvwz45bo4ibg]
        Target: [bafy2bzaceaayfun7qh5o3hccio7lzckpe3ue7w42pd76tmqrud5jh4srpmxys bafy2bzacebv4wvuuqo3btupedcwslsnd73p2clrvbxquijys5w6mpexozgtjw bafy2bzaceaxefbwye6xb3xwgbnab2hgkr5vzphjkn67putgswrjy62fh4coeg] (316424)
        Height diff:    77471
        Stage: header sync
        Height: 293597
        Elapsed: 6m51.820600417s
```

##### Sync wait

Use `sync wait` to output the state of your current chain as an ongoing process. This command may be used to wait on until the daemon is fully synchronized. 

```bash
lily sync wait
```

```bash
Worker: 0; Base: 0; Target: 414300 (diff: 414300)
State: header sync; Current Epoch: 410769; Todo: 3531
Validated 0 messages (0 per second)

```

## Jobs

Lily is capable of running Jobs, these include "Walk" and "Watch" jobs. Jobs can be controlled via the `job` command.

### Walk

A Walk job accepts a range of heights and traverses the chain from the heaviest tipset set at the upper height to the lower height using the parent state root present in each tipset.

### Watch

A Watch job follows the chain as is grows. Watch jobs subscribe to incoming tipsets and process them as they arrive. a confidence level may be specified which determines how many epochs lily should wait before processing a tipset. 

## Tasks

Lily provides several tasks to capture different aspects of the blockchain state. The type of data extracted extracted by Lily is controlled by the below tasks. Jobs accepts tasks to run as a comma separated list. The data extracted by a task is stored in its related Models.

|        Task         |                         Description                          |                            Models                            | Duration Per Tipset (Estimate) |
| :-----------------: | :----------------------------------------------------------: | :----------------------------------------------------------: | :----------------------------: |
|       blocks        |      Captures data about blocks and their relationships      | [block_headers](./models.md#block_headers), [block_parents](./models.md#block_parents), [drand_block_entries](./models.md#drand_block_entries) |              1 ms              |
|   chaineconomics    |           Captures circulating supply information.           |      [chain_economics](./models.md#drand_block_entries)      |             50 ms              |
|      consensus      | Captures consensus view of the chain which includes null rounds. |        [chain_consensus](./models.md#chain_consensus)        |              1 ms              |
|      messages       | Captures data about messages that were carried in a tipset's blocks. The receipt is also captured for any messages that were executed. Detailed information about gas usage by each messages as well as a summary of gas usage by all messages is also captured. The task does not produce any data until it has seen two tipsets since receipts are carried in the tipset following the one containing the messages. | [block_messages](./models.md#block_messages), [derived_gas_outputs](./models.md#derived_gas_outputs), [messages](./models.md#messages), [messages_gas_economy](./models.md#messages_gas_economy), [parsed_messages](./models.md#parsed_messages), [receipts](./models.md#receipts), |             50 ms              |
|   actorstatesraw    | Captures basic actor properties for any actors that have changed state and serializes a shallow form of the new state to JSON. | [actors](./models.md#actors), [actor_states](./models.md#actor_states) |             50 ms              |
|   actorstatesinit   | Captures changes to the init actor to provide mappings between canonical ID-addresses and actor addresses or public keys. |             [id_address](./models.md#id_address)             |              3 s               |
|  actorstatesmarket  | Captures new deal proposals and changes to deal states recorded by the storage market actor. | [market_deal_proposals](./models.md#market_deal_proposals), [market_deal_states](./models.md#market_deal_states) |              3 s               |
|  actorstatesminer   | Captures changes to miner actors to provide information about sectors, posts, and locked funds. | [miner_current_deadline_infos](./models.md#miner_current_deadline_infos), [miner_fee_debts](./models.md#miner_fee_debts), [miner_locked_funds](./models.md#miner_locked_funds), [miner_infos](./models.md#miner_infos), [miner_sector_posts](./models.md#miner_sector_posts), [miner_pre_commit_infos](./models.md#miner_pre_commit_infos), [miner_sector_infos](./models.md#miner_sector_infos),            [miner_sector_events](./models.md#miner_sector_events), [miner_sector_deals](./models.md#miner_sector_deals) |              18 s              |
| actorstatesmultisig |    Captures changes to multisig actors transaction state.    |  [multisig_transactions](./models.md#multisig_transactions)  |             100 ms             |
|  actorstatespower   | Captures changes to the power actors state including total power at each epoch and updates to the miner power claims. | [chain_powers](./models.md#chain_powers), [power_actor_claims](./models.md#power_actor_claims) |              5 s               |
|  actorstatesreward  | Captures changes to the reward actors state including information about miner rewards for each epoch. |           [chain_reward](./models.md#chain_reward)           |             100 ms             |
| actorstatesverifreg | Captures changes to the verified registry actor and verified registry clients. | [verified_registry_verified_clients](./models.md#verified_registry_verified_clients), [verified_registry_verifiers](./models.md#verified_registry_verifiers) |             10 ms              |
|  implicitmessages   | Captures internal message sends not appearing on chain including Multisig actor sends, Cron Event Ticks, Block and Reward Messages | [internal_messages](./models.md#internal_messages), [internal_parsed_messages](./models.md#internal_parsed_messages) |              1 ms              |
|     msapprovals     |        Captures approvals of multisig gated messages         |     [multisig_approvals](./models.md#multisig_approvals)     |             10 ms              |

