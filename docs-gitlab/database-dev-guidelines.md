# Database Development Guidelines

## Generic Interfaces

Although we're using the standard library [`database/sql`](https://golang.org/pkg/database/sql/), we should strive for
maintaining our internal service interfaces generic and decoupled from the underlying database client/driver. For this
reason, we should wrap standard types (like `sql.DB`), and their methods (only the ones we rely on) if they return
anything but an `error`. This will guarantee a low effort in case we want to swap the client/driver at some point.

## ER Model

The database ER model should be updated when doing a schema change. We're using
[pgModeler](https://github.com/pgmodeler/pgmodeler) for this (`.dbm` extension). The source files can be found at
`docs-gitlab/db`.
