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

lily daemon --repo=/var/lib/lily --config=/var/lib/lily/config.toml &

# wait for lily daemon
sleep 10

lily sync wait

tasks="actor_state,chain_power,miner_sector_event,market_deal_state,receipt,message,actor,miner_pre_commit_info,miner_sector_infos,miner_sector_infos_v7,miner_locked_fund,miner_fee_debt,parsed_message,block_message,derived_gas_outputs,block_parent,market_deal_proposal,internal_parsed_messages,internal_messages,block_header,miner_sector_deal,chain_consensus,chain_reward,chain_economics,miner_info,power_actor_claim,message_gas_economy,multisig_approvals,id_addresses"
lily job run --tasks=${tasks} --storage="Database1" watch notify --queue="Notifier1"

lily job run --tasks="peeragents" --storage="Database1" survey  --interval="24h"  notify --queue="Notifier1"

lily job run --storage="Database1" tipset-worker --queue="Worker1"

# resume daemon stdout
lily job wait --id=1
