# build/docker

### Background

Dockerfiles for lily have been broken into partials with a few
parameters. This was done in order to minimize repetitive configuration over
multiple files and to reduce the opportunity for human error.

### Usage

Dockerfile and Dockerfile.dev are designed to enable a lily image to be built
for any needed network. Dockerfile produces a minimal production image and
Dockerfile.dev provides additional tools and a full shell where troubleshooting
is easier.

Docker images can be made with `make docker-<network_target>[-dev]`. (Adding
`-dev` to the end of the make target makes the development version of that
docker image.)

`network_target` can be `mainnet`,`calibnet`, `interopnet`, `butterflynet`,
`nerpanet`, or `2k`

### Changes

Update templates in `build/docker/*` and run `make clean docker-files`.

### Pushing Tags

Images can be build and tagged with `make docker-<network_target>[-dev]-push`.
Docker registry credentials should already be configured.
