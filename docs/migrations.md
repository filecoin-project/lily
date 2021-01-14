# Schema Migrations

The database schema is versioned and every change requires a migration script to be executed. See [storage/migrations/README.md](storage/migrations/README.md) for more information.

## Checking current schema version

The visor `migrate` subcommand compares the **database schema version** to the **latest schema version** and reports any differences.
It also verifies that the **database schema** matches the requirements of the models used by visor. It is safe to run and will not alter the database.

Visor also verifies that the schema is compatible when the index or process subcommands are executed.

## Migrating schema to latest version

To migrate a database schema to the latest version, run:

    visor migrate --latest

Visor will only migrate a schema if it determines that it has exclusive access to the database. 

Visor can also be configured to automatically migrate the database when indexing or processing by passing the `--allow-schema-migration` flag.

## Reverting a schema migration

To revert to an earlier version, run:

    visor migrate --to <version>

**WARNING: reverting a migration is very likely to lose data in tables and columns that are not present in the earlier version**

