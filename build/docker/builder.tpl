# build/docker/builder.tpl
# partial for building visor without the entrypoint to allow for additional steps to be added

# ARG network_target determines which network the visor binary is built for.
# Default: mainnet
# Options: nerpanet, calibnet, butterflynet, interopnet, 2k
# See https://network.filecoin.io/ for more information about network_targets.
ARG network_target=mainnet
# ARG build_image is the image tag to use when building visor
ARG build_image

FROM $build_image AS builder

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
RUN make $network_target
RUN cp ./visor /usr/bin/

