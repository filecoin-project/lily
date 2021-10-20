#!/usr/bin/env bash

set -exo pipefail

mkdir -p ${LILY_REPO}/keystore

if [[ ! -z "${_LILY_DOCKER_INIT_IMPORT_MAINNET_SNAPSHOT}" ]]; then
  # import snapshot when _LILY_DOCKER_INIT_IMPORT_MAINNET_SNAPSHOT is set
  if [[ -f "${LILY_REPO}/datastore/_imported" ]]; then
    echo "Skipping import, found ${LILY_REPO}/datastore/_imported file."
  else
    echo "Importing snapshot from https://fil-chain-snapshots-fallback.s3.amazonaws.com/mainnet/minimal_finality_stateroots_latest.car..."
    lily init --import-snapshot="https://fil-chain-snapshots-fallback.s3.amazonaws.com/mainnet/minimal_finality_stateroots_latest.car"

    status=$?
    if [ $status -eq 0 ]; then
      touch "/var/lib/lily/datastore/_imported"
    fi
  fi
else
  # otherwise only init
  lily init
fi

chmod 0600 ${LILY_REPO}/keystore/*

lily $@
