### How to add a new vector
1. Create a CAR file for the chain snapshot being vectorized called "snapshot.car"
2. Create a CAR file for the genesis block of the snapshot called "genesis.car"
3. TAR the two files `tar -cvf <network>-<start_epoch>-<end_epoch>-fullstate.tar snapshot.car genesis.car`
4. Add the TAR file to IPFS (be sure its pinned somewhere, like web3.storage or estuary.tech)
5. Sha256 the TAR file `shasum -a 256 <network>-<start_epoch>-<end_epoch>-fullstate.tar`
6. Add an entry to `vectors.json`, the object key must not collide with an existing one:
```json
{
  "<network>-<start_epoch>-<end_epoch>-fullstate.tar": {
    "cid": "<cid_of_tar_file>",
    "network": "<network_snapshot_was_exported_from",
    "digest": "<sha256_of_tar_file_from_step_5>",
    "from": "<first_full_state_epoch_of_snapshot",
    "to": "<head_epoch_of_snapshot>"
  }
}
```