#!/usr/bin/env bash

set -exo pipefail

mkdir -p ${LILY_REPO}/keystore


if [[ ! -z "${LILY_DOCKER_INIT_IMPORT_MAINNET_SNAPSHOT}" ]]; then
  # set default snapshot path if not already defined
  snapshot="${LILY_DOCKER_INIT_IMPORT_SNAPSHOT_PATH:-https://snapshots.mainnet.filops.net/minimal/latest}"

  # import snapshot when LILY_DOCKER_INIT_IMPORT_MAINNET_SNAPSHOT is set
  if [[ -f "${LILY_REPO}/datastore/_imported" ]]; then
    echo "Skipping import, found ${LILY_REPO}/datastore/_imported file."
  else
    echo "Importing snapshot from ${snapshot}"
    lily init --import-snapshot=${snapshot}
    status=$?
    if [ $status -eq 0 ]; then
      touch "/var/lib/lily/datastore/_imported"
    fi
  fi
else
  # otherwise only init
  lily init
fi

chmod -R 0600 ${LILY_REPO}/keystore

lily $@
