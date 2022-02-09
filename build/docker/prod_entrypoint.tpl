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
