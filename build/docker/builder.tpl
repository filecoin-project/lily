# build/docker/builder.tpl
# partial for building visor without the entrypoint to allow for additional steps to be added

# ARG GO_BUILD_IMAGE is the image tag to use when building visor
ARG GO_BUILD_IMAGE
FROM $GO_BUILD_IMAGE AS builder

# ARG VISOR_NETWORK_TARGET determines which network the visor binary is built for.
# Options: mainnet, nerpanet, calibnet, butterflynet, interopnet, 2k
# See https://network.filecoin.io/ for more information about network_targets.
ARG VISOR_NETWORK_TARGET=mainnet

RUN apt-get update
RUN apt-get install -y \
  hwloc \
  jq \
  libhwloc-dev \
  mesa-opencl-icd \
  ocl-icd-opencl-dev

WORKDIR /go/src/github.com/filecoin-project/sentinel-visor
COPY . /go/src/github.com/filecoin-project/sentinel-visor

RUN make deps
RUN go mod download
RUN make $VISOR_NETWORK_TARGET
RUN cp ./visor /usr/bin/

