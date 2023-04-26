# Deployment in GCE
## Requirements
- docker compose
  - example: [install in debian](https://docs.docker.com/engine/install/debian/)
- git

## Provide the required credentials and configurations
- Rename **.env.example** to **.env** and provide the following env vars:
  - `LILY_REDIS_ADDR`
  - `LILY_REDIS_USERNAME`
  - `LILY_REDIS_PASSWORD`
  - `LILY_STORAGE_POSTGRESQL_DB_URL`
  - `PROMTAIL_ENV`: default value is `lily_gce_staging`
  - `PROMTAIL_REMOTE_URL`
  - `TRACING_REMOTE_URL`
  - `TRACING_REMOTE_USERNAME`
  - `TRACING_REMOTE_PASSWORD`
