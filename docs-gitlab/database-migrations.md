# Database Migrations

The PostgreSQL database schema migrations are managed through the `registry`
CLI. Internally, the registry is currently using the
[rubenv/sql-migrate](https://github.com/rubenv/sql-migrate) tool.

## Development

### Create Database

Please make sure to create a `registry_dev` and `registry_test` (naming
suggestion) database in your development PostgreSQL instance before running
migrations.
 
#### Example

```text
psql -h localhost -U postgres -w -c "CREATE DATABASE registry_dev;"
```

### Create New Migration

To create a new database migration run the following command from the root of
the repository:

```text
make new-migration [name]
```

A new migration file, named `[timestamp]_[name].go`, will be created under
`migrations/`. Migration files are prefixed with the timestamp of the current
system date formatted as `%Y%m%d%H%M%S`. 

Make sure to use a descriptive name for the migration, preferably denoting the
`action` and the object `name` and `type`. For example, to create a table `x`,
use `create_x_table`. To drop a column `a` from table `x`, use
`drop_x_a_column`. The name can only contain alphanumeric and underscore
characters.

New migrations are created based on a template, so we just need to fill in the
list of `Up` and `Down` SQL statements. All `Up` and `Down` statements are
executed within a transaction by default. To disable transactions you can set
the migration `DisableTransactionUp` and/or `DisableTransactionDown` attributes
to `true`.

#### Example

```text
$ make new-migration create_users_table
OK: ./migrations/20200713143615_create_users_table.go
```

## Administration

Database migrations are managed through the `registry` CLI, using the `database
migrate` command:

```text
$ registry database migrate --help config.yml
Usage:
  registry database migrate [flags]
  registry database migrate [command]

Available Commands:
  up          Apply up migrations
  down        Apply down migrations
  status      Show migration status
  version     Show current migration version

Flags:
  -h, --help   help for migrate

Use "registry database migrate [command] --help" for more information about a command.
```

### Pre-Requisites

* A PostgreSQL 11 (or higher) database for the registry must already exist;
* The database should be configured under the `database` section of the registry
  `config.yml` configuration file. Please see the [configuration
  docs](https://gitlab.com/gitlab-org/container-registry/-/blob/database/docs/configuration.md#database)
  for additional information;
* The `registry` binary, built from the source. See the [development
  guidelines](standalone-dev-registry.md) for more details.

### Apply Up Migrations

To apply pending up migrations use the `up` sub-command: 

```text
$ registry database migrate up --help config.yml
Apply up migrations

Usage:
  registry database migrate up [flags]

Flags:
  -d, --dry-run     do not commit changes to the database
  -h, --help        help for up
  -n, --limit int   limit the number of migrations (all by default)
```

If using the `--dry-run` flag, a migration plan (an ordered list of migrations
to apply) will be created and displayed but not executed. Additionally, it is
possible to limit the number of pending migration to apply using the `--limit`
flag. By default, all pending migrations are applied.

#### Example

```text
$ registry database migrate up -n 1 config.yml
20200713143615_create_users_table
OK: applied 1 migrations
```

### Apply Down Migrations

To apply pending down migrations (rollback) use the `down` sub-command: 

```text
$ registry database migrate down --help config.yml
Apply up migrations

Usage:
  registry database migrate up [flags]

Flags:
  -d, --dry-run     do not commit changes to the database
  -h, --help        help for up
  -n, --limit int   limit the number of migrations (all by default)
  -f, --force       no confirmation message
```

`--dry-run` and `--limit` flags also apply to the `down` command, and they work
as described for the `up` command.

Unlike the `up` command, for safety reasons, the `down` command requires
explicit user confirmation before applying the planned migrations. The `--force`
flag can be used to bypass the confirmation message.

#### Example

```text
$ registry database migrate down config.yml
20200713143615_create_users_table
20200527132906_create_repository_blobs_table
20200408193126_create_repository_manifest_lists_table
20200408192311_create_repository_manifests_table
20200319132237_create_tags_table
20200319132010_create_manifest_list_manifests_table
20200319131907_create_manifest_lists_table
20200319131632_create_manifest_layers_table
20200319131542_create_configurations_table
20200319131222_create_blobs_table
20200319130108_create_manifests_table
20200319122755_create_repositories_table
Preparing to apply down migrations. Are you sure? [y/N] y
OK: applied 12 migrations
```

### Status

The `status` sub-command displays a list of all migrations, including known (the
ones packaged in the executing `registry` binary) and unknown (the ones not
packaged in the `registry` binary but somehow applied in the database). The
applied timestamp (if any) is also displayed.

#### Example

```text
$ registry database migrate status config.yml
+--------------------------------------------------------+---------------------------------------+
|                       MIGRATION                        |                APPLIED                |
+--------------------------------------------------------+---------------------------------------+
| 20200319122755_create_repositories_table               | 2020-07-13 14:49:22.502491 +0100 WEST |
| 20200319130108_create_manifests_table                  | 2020-07-13 14:49:22.508141 +0100 WEST |
| 20200319131222_create_blobs_table                      | 2020-07-13 14:49:22.513534 +0100 WEST |
| 20200319131542_create_configurations_table             | 2020-07-13 14:49:22.53986 +0100 WEST  |
| 20200319131632_create_manifest_layers_table            | 2020-07-13 14:49:22.545756 +0100 WEST |
| 20200319131907_create_manifest_lists_table             | 2020-07-13 14:49:22.553113 +0100 WEST |
| 20200319132010_create_manifest_list_manifests_table    | 2020-07-13 14:49:22.585516 +0100 WEST |
| 20200319132237_create_tags_table                       | 2020-07-13 14:49:22.594482 +0100 WEST |
| 20200408192311_create_repository_manifests_table       | 2020-07-13 14:49:22.601056 +0100 WEST |
| 20200408193126_create_repository_manifest_lists_table  | 2020-07-13 14:49:22.613399 +0100 WEST |
| 20200527132906_create_repository_blobs_table (unknown) | 2020-07-13 14:49:22.639496 +0100 WEST |
| 20200713143615_create_users_table                      |                                       |
+--------------------------------------------------------+---------------------------------------+
```

### Version

The `version` sub-command displays the currently applied database migration.

#### Example

```text
$ registry database migrate version config.yml
20200527132906_create_repository_blobs_table
```
