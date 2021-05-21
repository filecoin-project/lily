# Schema Migrations

This directory holds migration scripts for migrating the database schema used by Visor.

The database schema is versioned using a simple monotonic counter.

Migrations are Go source files named after the schema version they migrate to plus a short tag. 
For example,`2_visor_initial.go` is the migration to version 2 of the schema. 

Each migration consists of an `init()` function with a single call to `MustRegisterTx` with two arguments: a set of *up* statements that migrate the schema up from the previous version and a set of *down* statements that migrate back down to the previous version. The *down* argument is optional but if omitted it prevents automatic rollback of failed deployments.

Raw SQL migrations may also be used. See the [go-pg migrations documentation](https://github.com/go-pg/migrations#sql-migrations) for more details.

The **latest schema version** is defined to be the highest schema version used by a migration in this directory. 

The schema in use by the database is held in a table called `gopg_migrations`. This records a complete history of the schema including both up and down migrations. The most recent entry in this table is known as the **database schema version**

Every running instance of Visor now takes a shared advisory lock in the database to indicate their presence. A schema migration requires an exclusive advisory lock which will fail if there are any existing locks already taken. This ensures that migrations are only performed by a single instance of Visor.

## When to write a schema migration

A schema migration is required any time:

 - a new model is added
 - a model changes name
 - a field is added to a model
 - a field is renamed
 - a field changes its type

It is recommended to also create a migration for deleted models and fields but this is not strictly necessary.

## How to write a schema migration

The next schema version will be one higher than the highest schema version listed in this directory. Every migration, no matter how small, increments the schema version.

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


