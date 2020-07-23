package datastore

import (
	"context"
	"database/sql"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
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
	log *statementLogger
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	reportTime := db.log.statement(query, args...)
	defer reportTime()

	return db.DB.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	reportTime := db.log.statement(query, args...)
	defer reportTime()

	return db.DB.QueryRowContext(ctx, query, args...)
}

func (db *DB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	reportTime := db.log.statement(query)
	defer reportTime()

	return db.DB.PrepareContext(ctx, query)
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	reportTime := db.log.statement(query)
	defer reportTime()

	return db.DB.ExecContext(ctx, query, args...)
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

// statementLogger allows queries to be pretty printed in debug mode without
// incuring a significant performance penalty in production environments.
type statementLogger struct {
	*logrus.Entry
}

// statement logs the statement and its args returning a function which may
// be deferred to log the execution time of the query.
func (sl *statementLogger) statement(statement string, args ...interface{}) func() {
	if !sl.Logger.IsLevelEnabled(logrus.DebugLevel) {
		return func() {}
	}

	s := regexp.MustCompile(`\s+|\t+|\n+`).ReplaceAllString(statement, " ")

	l := sl.WithFields(logrus.Fields{"statement": s, "args": args})
	l.Debug("begin query")

	start := time.Now()
	return func() {
		l.WithFields(logrus.Fields{"duration_us": time.Since(start).Microseconds()}).Debug("end query")
	}
}

// Open opens a database.
func Open(dsn *DSN, log ...*logrus.Entry) (*DB, error) {
	db, err := sql.Open(driverName, dsn.String())
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	var l *logrus.Entry

	if len(log) != 0 && log[0] != nil {
		l = log[0]
	} else {
		lg := logrus.New()
		lg.SetOutput(ioutil.Discard)
		l = logrus.NewEntry(lg)
	}

	return &DB{db, dsn, &statementLogger{l}}, nil
}
