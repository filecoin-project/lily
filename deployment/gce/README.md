# Deployment in GCE
## Requirements
- docker compose
  - example: [install in debian](https://docs.docker.com/engine/install/debian/)
- git

## Provide the necessary URL and Password
- In **lily/worker.env**, **lily/notifier.env**
  - `LILY_REDIS_ADDR`
  - `LILY_REDIS_PASSWORD`
- In **lily/worker_config.toml**, **lily/notifier_config.toml**
  - `[Storage.Postgresql.Database1]`: to replace the `POSTGRESQL_URL` 
  - `[Queue.Notifiers.Notifier1]`: to replace the `REDIS_ADDRESS`
- In **promtail/config.yml**
  - `clients:url`: to replace the `remote_loki_url`