# Database Development Guidelines

## Generic Interfaces

Although we're using the standard library
[`database/sql`](https://golang.org/pkg/database/sql/), we should strive for
maintaining our internal service interfaces generic and decoupled from the
underlying database client/driver. For this reason, we should wrap standard
types (like `sql.DB`), and their methods (only the ones we rely on) if they
return anything but an `error`. This will guarantee a low effort in case we want
to swap the client/driver at some point.

## ER Model

The database ER model should be updated when doing a schema change. We're using
[pgModeler](https://github.com/pgmodeler/pgmodeler) for this (`.dbm` extension).
The source files can be found at `docs-gitlab/db`.

## Naming Conventions

Although there are no specific conventions that we should follow at GitLab for
naming database indexes and constraints, we decided to adopt the following one
to improve consistency and discoverability:

| Type                      | Syntax                                                | Example                                         |
| ------------------------- | ----------------------------------------------------- | ----------------------------------------------- |
| `PRIMARY KEY` constraints | `pk_<table name>`                                     | `pk_repositories`                               |
| `FOREIGN KEY` constraints | `fk_<table name>_<column name>_<referred table name>` | `fk_repository_manifests_manifest_id_manifests` |
| `UNIQUE` constraints      | `uq_<table name>_<column(s) name>`                    | `uq_manifest_layers_manifest_id_blob_id`       |
| `CHECK` constraints       | `ck_<table name>_<column(s) name>_<validation name>`  | `ck_layers_media_type_length`                   |
| Indexes                   | `ix_<table name>_<column(s) name>`                    | `ix_tags_manifest_id`                           |

## SQL Formatting

Long, complex or multi-line SQL statements must be formatted with
[pgFormatter](https://github.com/darold/pgFormatter), using the default settings.
There are plugins for several editors/IDEs and there is also an online version at
[sqlformat.darold.net](http://sqlformat.darold.net/).

## Testing

### Golden Files

Some database operations generate a considerable amount of data on the database.
In some cases it's not practical or maintainable to define structs for all
expected rows and then compare them one by one.

When the only thing we need to assert is that the database data ended up in a
given state, we can use golden files, which should contain a pre-validated
snapshot/dump (in JSON format for easy readability) of a given table.

Therefore, instead of comparing item by item, we can save the expected database
table content to a file named `<table>.golden` within
`<package>/testdata/golden/<test function name>`, and then compare these against
an actual JSON dump in the test functions.

#### Test Flags

To facilitate the development process, two flags for the `go test` command were
added:

- `update`: Updates existing golden files with a new expected value. For
  example, if we change a column name, it's impractical to update the golden
  files manually. With this command the golden files are automatically updated
  with a fresh dump that reflects the new column name.

- `create`: Create missing golden files, followed by `update`. In case we add
  new tables, new golden files need to be created for them. Instead of creating
  them manually, we can use this flag and they will be automatically created and
  updated with the current table content.

These flags are defined in `registry.datastore.datastore_test`. The caller
(running test) is responsible for passing the value of these flags to the
`registry.datastore.testutil.CompareWithGoldenFile(tb testing.TB, path string,
actual []byte, create, update bool)` helper function. Whenever this function is
called, it'll attempt to find the golden file at `path`, read it and then
compare its contents against `actual`. If these don't match, the test will fail
and a diff will be presented in the test output.

#### Example

Please have a look at the integration tests for `registry.datastore.Importer`.
We'll use this test as example.

##### Create

If we wanted to test a new table `foo` in `TestImporter_Import`, we would do:

1. Run the `go test` command with the `create` flag:

```
go test -v -tags=integration github.com/docker/distribution/registry/datastore -create
```

After doing so, a new
`registry/datastore/testdata/golden/TestImporter_Import/foo.golden` golden file
would be created with the contents of the `foo` table during the test. We could
then modify this file to match what should be the correct/expected state of the
`foo` table for our test.

Alternatively, we could also manually create the golden file in the path
mentioned above and skip the `create` flag.

##### Update

If for example we changed the name of column `a` in the `foo` table to `b`,
instead of manually updating the corresponding golden file we could simply rerun
the test with the `update` flag:

```
go test -v -tags=integration github.com/docker/distribution/registry/datastore -update
```

Doing so, the golden file would be updated with the current state of the `foo`
table at the time of running the test (which would already have the column `a`
renamed to `b`).

Alternatively, we could update the golden file manually as well. This update
flag is useful when whe changed something on our database and we're sure that
the new output is the correct one, so we can simply use it to refresh all golden
files to match the new expectation.