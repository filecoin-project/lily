# Test Vectors
This directory contains the test vector manuscript.
Add new test vectors by appending them to `vectors.json`.
Execute `make itest-<network_name>` to download and execute vector tests.
The contents of the `vectors.json` must contain the following:
- `"file_name"`
  - `cid`: The CID of a car file produced by running a chain export.
  - `network`: The network the CAR file was exported from.
  - `digest`: Sha256 of CAR file, eg. `shasum -a 256 chain.car`.
  - `from`: earliest full state epoch.
  - `to`: latest full state epoch.

Vectors are downloaded to `LILY_TEST_VECTORS` defaulting to `/var/temp/lily-test-vectors` based on network name.