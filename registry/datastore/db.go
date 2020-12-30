package datastore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
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

// BeginTx wraps sql.Tx from the innner sql.DB within a datastore.Tx.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)

	return &Tx{tx, db.log}, err
}

// Begin wraps sql.Tx from the inner sql.DB within a datastore.Tx.
func (db *DB) Begin() (*Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

// Tx is a database transaction that implements Querier.
type Tx struct {
	*sql.Tx
	log *statementLogger
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	reportTime := tx.log.statement(query, args...)
	defer reportTime()

	return tx.Tx.QueryContext(ctx, query, args...)
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	reportTime := tx.log.statement(query, args...)
	defer reportTime()

	return tx.Tx.QueryRowContext(ctx, query, args...)
}

func (tx *Tx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	reportTime := tx.log.statement(query)
	defer reportTime()

	return tx.Tx.PrepareContext(ctx, query)
}

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	reportTime := tx.log.statement(query, args...)
	defer reportTime()

	return tx.Tx.ExecContext(ctx, query, args...)
}

// DSN represents the Data Source Name parameters for a DB connection.
type DSN struct {
	Host           string
	Port           int
	User           string
	Password       string
	DBName         string
	SSLMode        string
	SSLCert        string
	SSLKey         string
	SSLRootCert    string
	ConnectTimeout time.Duration
}

// String builds the string representation of a DSN.
func (dsn *DSN) String() string {
	var params []string

	port := ""
	if dsn.Port > 0 {
		port = strconv.Itoa(dsn.Port)
	}
	connectTimeout := ""
	if dsn.ConnectTimeout > 0 {
		connectTimeout = fmt.Sprintf("%.0f", dsn.ConnectTimeout.Seconds())
	}

	for _, param := range []struct{ k, v string }{
		{"host", dsn.Host},
		{"port", port},
		{"user", dsn.User},
		{"password", dsn.Password},
		{"dbname", dsn.DBName},
		{"sslmode", dsn.SSLMode},
		{"sslcert", dsn.SSLCert},
		{"sslkey", dsn.SSLKey},
		{"sslrootcert", dsn.SSLRootCert},
		{"connect_timeout", connectTimeout},
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

// Address returns the host:port segment of a DSN.
func (dsn *DSN) Address() string {
	return net.JoinHostPort(dsn.Host, strconv.Itoa(dsn.Port))
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

	a := make([]interface{}, len(args))
	whitespace := regexp.MustCompile(`\s+|\t+|\n+`)

	// Copy args to prevent mutating the real payload as it is formatted.
	for i := range args {
		a[i] = args[i]

		if payload, ok := a[i].(json.RawMessage); ok {
			a[i] = whitespace.ReplaceAllString(string(payload), " ")
		}
	}

	s := whitespace.ReplaceAllString(statement, " ")
	l := sl.WithFields(logrus.Fields{"statement": s, "args": a})
	start := time.Now()

	return func() {
		l.WithFields(logrus.Fields{"duration_us": time.Since(start).Microseconds()}).Debug("query")
	}
}

type openOpts struct {
	logger *logrus.Entry
	pool   *PoolConfig
}

type PoolConfig struct {
	MaxIdle     int
	MaxOpen     int
	MaxLifetime time.Duration
}

// OpenOption is used to pass options to Open.
type OpenOption func(*openOpts)

// WithLogger configures the logger for the database connection handler.
func WithLogger(l *logrus.Entry) OpenOption {
	return func(opts *openOpts) {
		opts.logger = l
	}
}

// WithPoolConfig configures the settings for the database connection pool.
func WithPoolConfig(c *PoolConfig) OpenOption {
	return func(opts *openOpts) {
		opts.pool = c
	}
}

var defaultLogger = logrus.New()

func applyOptions(opts []OpenOption) openOpts {
	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	config := openOpts{
		logger: logrus.NewEntry(log),
		pool:   &PoolConfig{},
	}

	for _, v := range opts {
		v(&config)
	}

	return config
}

// Open creates a database connection handler.
func Open(dsn *DSN, opts ...OpenOption) (*DB, error) {
	config := applyOptions(opts)

	db, err := sql.Open(driverName, dsn.String())
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(config.pool.MaxOpen)
	db.SetMaxIdleConns(config.pool.MaxIdle)
	db.SetConnMaxLifetime(config.pool.MaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db, dsn, &statementLogger{config.logger}}, nil
}
