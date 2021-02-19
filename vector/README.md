# Visor Vector How-To
You can use the `visor vector` command to create and execute test vectors with Visor. Vectors serve as a set of reproducible scenarios that allow developers to ensure their changes haven't broken Visor. All that's required is a Lotus repo.

## How to build a vector
To build a test vector, use the `vector build` command. The example below creates a test vector file named "blocks_471000-471010.json".
```
visor --repo-read-only=false --repo="~/.lotus" vector build --tasks=blocks --from=471000 --to=471010 --vector-file="./blocks_471000-471010.json" --vector-desc="block models from epoch 471000 to 471010"
```
The file contains information about the version of Visor used to build it:
```
$ cat blocks_471000-471010.json | jq ".metadata"
{
 "version": "v0.4.0+rc2-55-g17c8329-dirty",
 "description": "block models from epoch 471000 to 471010",
 "network": "mainnet",
 "time": 1613515320
}
```
the commands to produce it:
```
$ cat blocks_471000-471010.json | jq ".parameters"
{
 "from": 471000,
 "to": 471010,
 "tasks": [
   "blocks"
 ],
 "actor-address": ""
}
```
the extracted chain state as a base64 encoded CAR file:
```
$ cat blocks_471000-471010.json | jq ".car"
<lot_of_data>
```
and the models extracted:
```
$ cat blocks_471000-471010.json | jq ".expected[].block_headers"
[
 {
   "Height": 471010,
   "Cid": "bafy2bzacecq2fysiktcuc5r762hvuxstwv4qzx645ld4h66spenheckyuko2y",
   "Miner": "f02770",
   "ParentWeight": "10230517226",
   "ParentBaseFee": "3275882399",
   "ParentStateRoot": "bafy2bzacecsckqgj6ufbefj7bx6odzvvbftky77py5x2th3wrqx3yvpksyuc6",
   "WinCount": 1,
   "Timestamp": 1612436700,
   "ForkSignaling": 0
 },
 {
   "Height": 471010,
   "Cid": "bafy2bzacecxmra4cdbqq7lxao7e2gmxvf7ef3o464et7ejykcn24itps7vnau",
   "Miner": "f02775",
   "ParentWeight": "10230517226",
   "ParentBaseFee": "3275882399",
   "ParentStateRoot": "bafy2bzacecsckqgj6ufbefj7bx6odzvvbftky77py5x2th3wrqx3yvpksyuc6",
   "WinCount": 1,
   "Timestamp": 1612436700,
   "ForkSignaling": 0
 },
 ...
```

## How to execute a vector
To execute a vector, use the `vector execute` command. The example below executes a test vector file named "blocks_471000-471010.json".
```
$ ./visor vector execute --vector-file=blocks_471000-471010.json
2021-02-16T16:32:29.225-0800   INFO   vector vector/runner.go:146 Validate Model block_headers: Passed
2021-02-16T16:32:29.230-0800   INFO   vector vector/runner.go:146 Validate Model block_parents: Passed
2021-02-16T16:32:29.230-0800   INFO   vector vector/runner.go:146 Validate Model drand_block_entries: Passed
```
When executing the vector file Visor uses the data in the `car` field as its data source, executes the commands in the `parameters` field, then validates the data returned from the execution matches the `expected` models in the vector file.

## How to save a vector
Vector files are stored on the IPFS network, and a list of their hashes is kept in the `VECTOR_MANIFEST` file. Storing the hashes in the git repo and the files in IPFS helps keep our repo compact. The example below will demonstrate how to save a vector file to run as a part of CI. We will use the aforementioned `blocks_471000-471010.json` in this example.
```
$ ipfs add blocks_471000-471010.json
added QmPTKSr1uwExGTVByDUoVm4CfrjU34oNV6gfvWvufaN9h1 blocks_471000-471010.json
 146.43 KiB / 146.43 KiB [==] 100.00%
```
next, add the hash to the manifest:
```
$ echo QmPTKSr1uwExGTVByDUoVm4CfrjU34oNV6gfvWvufaN9h1 >> ./vector/VECTOR_MANIFEST
```
then make the vector deps:
```
$ make deps
cd ./vector; ./fetch_vectors.sh
Fetching QmPTKSr1uwExGTVByDUoVm4CfrjU34oNV6gfvWvufaN9h1
 ```
You can see the vector file is in `vectors/data/`:
```
ls vector/data/
QmPTKSr1uwExGTVByDUoVm4CfrjU34oNV6gfvWvufaN9h1_block_models_from_epoch_471000_to_471010.json
```
Finally, execute the vector unit tests:
```
$ go test -v ./vector/...
=== RUN  TestExecuteVectors
=== RUN  TestExecuteVectors/QmPTKSr1uwExGTVByDUoVm4CfrjU34oNV6gfvWvufaN9h1_block_models_from_epoch_471000_to_471010.json
--- PASS: TestExecuteVectors (65.08s)
   --- PASS: TestExecuteVectors/QmPTKSr1uwExGTVByDUoVm4CfrjU34oNV6gfvWvufaN9h1_block_models_from_epoch_471000_to_471010.json (0.02s)
```
Push your changes to `VECTOR_MANIFEST` up execution in CI.





