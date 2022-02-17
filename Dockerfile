# This file was generated with `make docker-files` and should not
# be editted directly. Please see build/docker/README.md for more info.

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

# build/docker/prod_entrypoint.tpl
# partial for producing a minimal image by extracting the binary
# from a prior build step (builder.tpl)

FROM buildpack-deps:bookworm-curl
COPY --from=builder /go/src/github.com/filecoin-project/lily/lily /usr/bin/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libOpenCL.so* /lib/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libhwloc.so* /lib/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libnuma.so* /lib/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libltdl.so* /lib/

RUN apt-get update
RUN apt-get install -y --no-install-recommends \
      jq

ENTRYPOINT ["/usr/bin/lily"]
CMD ["--help"]
