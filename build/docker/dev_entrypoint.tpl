# build/docker/dev_entrypoint.tpl
# partial for completing a dev lily dockerfile

ENTRYPOINT ["/usr/bin/lily"]
CMD ["--help"]
