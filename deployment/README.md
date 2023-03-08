# Deployment in GCE
## Requirements
- docker compose
  - example: [install in debian](https://docs.docker.com/engine/install/debian/)
- git

## Provide the necessary URL and Password
- In **lily/lily.env**
  - `LILY_REDIS_ADDR`
  - `LILY_REDIS_PASSWORD`
  - `LILY_STORAGE_POSTGRESQL_DB_URL`
- In **lily/config.toml**
  - `[Storage.Postgresql.Database1]`: to replace the `POSTGRESQL_URL` 
  - `[Queue.Notifiers.Notifier1]`: to replace the `REDIS_ADDRESS`
- In **promtail/promtail.env**
  - `REMOTE_LOKI_URL`
  - `LILY_ENV`: default value is `lily_gce_staging`
