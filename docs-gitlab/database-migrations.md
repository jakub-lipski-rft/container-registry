# Database Migrations

The registry uses a PostgreSQL database and migrations are managed with the 
[migrate](https://github.com/golang-migrate/migrate) tool.

## Install `migrate` tool

Make sure you have the dev tools installed by running:

```
./script/setup/install-dev-tools
```

## Create

To create a database migration run the following command:

```
migrate create -ext sql -dir db/migrations <action>_<name>_<type>
```

Make sure to use a descriptive name for the migration, denoting the `action` and the object `name` and its `type`. For 
example, to create a table `x`, use `create_x_table`. To drop a column `a` from table `x`, use `drop_x_a_column`.

Migration files are prefixed with a timestamp to ensure they're in the right sequence. An `up` and `down` migration will
be created under `db/migrations`. Make sure to open and fill each of them. 

For general best practices, please look at the 
[`migrate` documentation](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md).

## Run

If this is the first time you're running migrations, please make sure you create a `registry` database in your 
PostgreSQL server. Example:

```
psql -h localhost -U postgres -w -c "create database registry;"
```

Please look at the [`migrate` CLI docs](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) for further
instructions.