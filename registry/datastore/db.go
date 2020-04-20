package datastore

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/docker/distribution/migrations"
	bindata "github.com/golang-migrate/migrate/source/go_bindata"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/lib/pq"
)

type migrateDirection int

const (
	driverName = "postgres"

	up migrateDirection = iota
	down
)

// Queryer is the common interface to execute queries on a database.
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// DB is a database handle that implements Querier.
type DB struct {
	*sql.DB
	dsn *DSN
}

// Tx is a database transaction that implements Querier.
type Tx struct {
	*sql.Tx
}

// DSN represents the Data Source Name parameters for a DB connection.
type DSN struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// String builds the string representation of a DSN.
func (dsn *DSN) String() string {
	var params []string

	port := ""
	if dsn.Port > 0 {
		port = strconv.Itoa(dsn.Port)
	}

	for _, param := range []struct{ k, v string }{
		{"host", dsn.Host},
		{"port", port},
		{"user", dsn.User},
		{"password", dsn.Password},
		{"dbname", dsn.DBName},
		{"sslmode", dsn.SSLMode},
	} {
		if len(param.v) == 0 {
			continue
		}

		param.v = strings.ReplaceAll(param.v, "'", `\'`)
		param.v = strings.ReplaceAll(param.v, " ", `\ `)

		params = append(params, param.k+"="+param.v)
	}

	return strings.Join(params, " ")
}

// Open opens a database.
func Open(dsn *DSN) (*DB, error) {
	db, err := sql.Open(driverName, dsn.String())
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db, dsn}, nil
}

// MigrateUp applies all up migrations.
func (db *DB) MigrateUp() error {
	return db.migrate(up)
}

// MigrateDown applies all down migrations.
func (db *DB) MigrateDown() error {
	return db.migrate(down)
}

// MigrateVersion returns the current migration version.
func (db *DB) MigrateVersion() (int, error) {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return 0, err
	}

	v, _, err := driver.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, err
	}

	return v, nil
}

func (db *DB) migrate(direction migrateDirection) error {
	destDriver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return err
	}

	src := bindata.Resource(migrations.AssetNames(), migrations.Asset)
	srcDriver, err := bindata.WithInstance(src)
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("go-bindata", srcDriver, db.dsn.DBName, destDriver)
	if err != nil {
		return err
	}

	if direction == up {
		err = m.Up()
	} else {
		err = m.Down()
	}

	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
