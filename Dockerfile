# Builder
FROM golang:1.15.2 as builder

# Install deps for filecoin-project/filecoin-ffi
RUN apt-get update
RUN apt-get install -y jq mesa-opencl-icd ocl-icd-opencl-dev

WORKDIR /go/src/github.com/filecoin-project/sentinel-visor
COPY . /go/src/github.com/filecoin-project/sentinel-visor
RUN make deps && make build

# Runner
FROM buildpack-deps:buster-curl
# Grab the things
COPY --from=builder /go/src/github.com/filecoin-project/sentinel-visor/sentinel-visor /usr/bin/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libOpenCL.so* /lib/

ENTRYPOINT ["/usr/bin/sentinel-visor"]
CMD ["--help"]