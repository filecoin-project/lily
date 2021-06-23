# This file was generated with `make docker-files` and should not
# be editted directly. Please see build/docker/README.md for more info.

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
RUN go mod download -x
RUN make $VISOR_NETWORK_TARGET
RUN cp ./visor /usr/bin/

# build/docker/prod_entrypoint.tpl
# partial for producing a minimal image by extracting the binary
# from a prior build step (builder.tpl)

FROM buildpack-deps:buster-curl
COPY --from=builder /go/src/github.com/filecoin-project/sentinel-visor/visor /usr/bin/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libOpenCL.so* /lib/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libhwloc.so* /lib/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libnuma.so* /lib/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libltdl.so* /lib/

ENTRYPOINT ["/usr/bin/visor"]
CMD ["--help"]
