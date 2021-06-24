# build/docker/dev_entrypoint.tpl
# partial for completing a dev visor dockerfile

ENTRYPOINT ["/usr/bin/visor"]
CMD ["--help"]
