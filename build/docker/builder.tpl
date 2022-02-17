# build/docker/builder.tpl
# partial for building lily without the entrypoint to allow for additional steps to be added

# ARG GO_BUILD_IMAGE is the image tag to use when building lily
ARG GO_BUILD_IMAGE

FROM $GO_BUILD_IMAGE AS builder

# ARG LILY_NETWORK_TARGET determines which network the lily binary is built for.
# Options: mainnet, nerpanet, calibnet, butterflynet, interopnet, 2k
# See https://network.filecoin.io/ for more information about network_targets.
ARG LILY_NETWORK_TARGET
ENV LILY_NETWORK_TARGET=$LILY_NETWORK_TARGET

RUN apt-get update
RUN apt-get install -y \
  hwloc \
  jq \
  libhwloc-dev \
  mesa-opencl-icd \
  ocl-icd-opencl-dev

WORKDIR /go/src/github.com/filecoin-project/lily
COPY . /go/src/github.com/filecoin-project/lily

RUN make deps
RUN go mod download

# ARG LILY_VERSION will set the binary version upon build
ARG LILY_VERSION
ENV LILY_VERSION=$LILY_VERSION
RUN make $LILY_NETWORK_TARGET
RUN cp ./lily /usr/bin/

