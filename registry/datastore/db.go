package datastore

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

const driverName = "postgres"

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
