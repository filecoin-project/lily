# build/docker/dev_entrypoint.tpl
# partial for completing a dev lily dockerfile

RUN apt-get update
RUN apt-get install -y --no-install-recommends \
      jq

ENTRYPOINT ["/usr/bin/lily"]
CMD ["--help"]
