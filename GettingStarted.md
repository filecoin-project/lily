# Getting Started

This document aims to quickly get you started using lily to process the Filecoin blockchain.

## Background

Lily is a wrapper around a Lotus node including extra logic that allows it to inspect the
state of a tipsets as well as diff a child and parent tipset and determine what has changed.
Lily can then persist the extracted state to a database or a file.
Lily has many of the same system dependencies as lotus so go ahead and follow the dependency installation
process described [here](https://docs.filecoin.io/get-started/lotus/installation/#linux).

## Initialize Lily

Start by initializing lily with a complete chain snapshot:
```shell
$ lily init --import-snapshot https://fil-chain-snapshots-fallback.s3.amazonaws.com/mainnet/complete_chain_with_finality_stateroots_latest.car
```

