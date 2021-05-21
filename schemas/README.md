# Schema Definitions

This directory and its subdirectories holds the scripts for creating TimescaleDB database schemas compatible with the
visor data model defined by the strucs in the `models` package.

The data model and the corresponding database schemas are versioned using a `major` version number plus a `patc` number.
For example `0.28`. Schemas with different major versions are not compatible with one another. Manual data migration 
is required to transition from one major version to another.

Patches are applied on top of the base schema for a major version and contain only additive, non-breaking changes with no data migration.
This ensures pPatches are safe and can be applied by visor automatically. Some examples of additive, non-migration patches are adding a 
new table or view, adding field comments or adding a nullable column with a default. 

Changes that are not suitable for patches include adding an index or changing a column type (may require long migrations and 
database unavailability), removing a table, renaming a table or column.

## Schema Directories

Each major version of the database schema is contained in its own Go package in a subdirectory prefixed with `v`, for example `v0`, `v1`, `v2` etc.

This package must export a string variable called `Base` which contains the base sql that is executed when the schema is being created initially. 
The package must also register its major version number by calling `schemas.RegisterSchema` which allows Visor to detect the most recent schema
available.

Additionally the package may contain migration scripts for applying patches onto the base schema over time. The migration scripts are registered 
with a collection called `Patches` which must be exported.

Migrations are Go source files named after the patch version they migrate to plus a short tag. 
For example,`2_visor_initial.go` is the migration to patch 2 of the schema. 

Each migration consists of an `init()` function with a single call to `Patches.MustRegisterTx` with up to two arguments: 

 - a set of *up* statements that migrate the schema up from the previous version 
 - a set of *down* statements that migrate back down to the previous version. The *down* argument is optional but if omitted it prevents automatic rollback of failed deployments.

The **latest patch version** is defined to be the highest version used by a migration in this directory. 

The patch in use by the database is held in a table called `gopg_migrations`. This records a complete history of the schema including both up and down migrations. The most recent entry in this table is known as the **schema patch version**

The major version is 0 unless a table called `visor_version` exists with a single row containing the major version number. A new major schema should create and populate this table in its base SQL.

Every running instance of Visor now takes a shared advisory lock in the database to indicate their presence. A schema migration requires an exclusive advisory lock which will fail if there are any existing locks already taken. This ensures that migrations are only performed by a single instance of Visor.

## When to write a schema migration

A schema migration is required any time:

 - a new model is added
 - a new view is added
 - a field is added to a model

## How to write a schema migration

The next patch version will be one higher than the highest patch version listed in this directory. Every migration, no matter how small, increments the patch version.

**Do not modify existing migration files**

1. Make the required changes to the Go models. 
2. Ensure your test database is on the latest schema version (run `sentinel-visor migrate --latest`)
3. Run the `TestSchemaIsCurrent` test in the storage package to compare the models to the current database schema. This will log the `CREATE TABLE` ddl for any altered tables.
4. Create a new migration file using the next schema version as a prefix.
5. Add all the statements required to migrate the schema from the previous version to an `up` variable.
6. Add all the statements required to migrate the schema to the previous version to a `down` variable.
7. Call `Patches.MustRegisterTx` with your `up` and `down` statements.
8. Run the migration by running `sentinel-visor migrate --to <new-version>`
9. Test the migration is compatible by using `sentinel-visor migrate`
10. If needed, revert the migration by running `sentinel-visor migrate --to <old-version>`


