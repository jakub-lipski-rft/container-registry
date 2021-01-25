package datastore

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/distribution/configuration"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/sirupsen/logrus"
)

const driverName = "pgx"

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

// BeginTx wraps sql.Tx from the innner sql.DB within a datastore.Tx.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)

	return &Tx{tx}, err
}

// Begin wraps sql.Tx from the inner sql.DB within a datastore.Tx.
func (db *DB) Begin() (*Tx, error) {
	return db.BeginTx(context.Background(), nil)
}

// Tx is a database transaction that implements Querier.
type Tx struct {
	*sql.Tx
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

type openOpts struct {
	logger   *logrus.Entry
	logLevel pgx.LogLevel
	pool     *PoolConfig
}

type PoolConfig struct {
	MaxIdle     int
	MaxOpen     int
	MaxLifetime time.Duration
}

// OpenOption is used to pass options to Open.
type OpenOption func(*openOpts)

// WithLogger configures the logger for the database connection driver.
func WithLogger(l *logrus.Entry) OpenOption {
	return func(opts *openOpts) {
		opts.logger = l
	}
}

// WithLogLevel configures the logger level for the database connection driver.
func WithLogLevel(l configuration.Loglevel) OpenOption {
	var lvl pgx.LogLevel
	switch l {
	case configuration.LogLevelTrace:
		lvl = pgx.LogLevelTrace
	case configuration.LogLevelDebug:
		lvl = pgx.LogLevelDebug
	case configuration.LogLevelInfo:
		lvl = pgx.LogLevelInfo
	case configuration.LogLevelWarn:
		lvl = pgx.LogLevelWarn
	default:
		lvl = pgx.LogLevelError
	}

	return func(opts *openOpts) {
		opts.logLevel = lvl
	}
}

// WithPoolConfig configures the settings for the database connection pool.
func WithPoolConfig(c *PoolConfig) OpenOption {
	return func(opts *openOpts) {
		opts.pool = c
	}
}

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

type logger struct {
	*logrus.Entry
}

// used to minify SQL statements on log entries by removing multiple spaces, tabs and new lines.
var logMinifyPattern = regexp.MustCompile(`\s+|\t+|\n+`)

// Log implements the pgx.Logger interface.
func (l *logger) Log(_ context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	// silence if debug level is not enabled, unless it's a warn or error
	if !l.Logger.IsLevelEnabled(logrus.DebugLevel) && level != pgx.LogLevelWarn && level != pgx.LogLevelError {
		return
	}
	var log *logrus.Entry
	if data != nil {
		// minify SQL statement, if any
		if _, ok := data["sql"]; ok {
			raw := fmt.Sprintf("%v", data["sql"])
			data["sql"] = logMinifyPattern.ReplaceAllString(raw, " ")
		}
		// use milliseconds for query duration
		if _, ok := data["time"]; ok {
			raw := fmt.Sprintf("%v", data["time"])
			d, err := time.ParseDuration(raw)
			if err == nil { // this should never happen, but lets make sure to avoid panics and missing log entries
				data["duration_ms"] = d.Milliseconds()
				delete(data, "time")
			}
		}
		// convert known keys to snake_case notation for consistency
		if _, ok := data["rowCount"]; ok {
			data["row_count"] = data["rowCount"]
			delete(data, "rowCount")
		}
		log = l.WithFields(data)
	} else {
		log = l.Entry
	}

	switch level {
	case pgx.LogLevelTrace:
		log.Trace(msg)
	case pgx.LogLevelDebug:
		log.Debug(msg)
	case pgx.LogLevelInfo:
		log.Info(msg)
	case pgx.LogLevelWarn:
		log.Warn(msg)
	case pgx.LogLevelError:
		log.Error(msg)
	default:
		// this should never happen, but if it does, something went wrong and we need to notice it
		log.WithField("invalid_log_level", level).Error(msg)
	}
}

// Open creates a database connection handler.
func Open(dsn *DSN, opts ...OpenOption) (*DB, error) {
	config := applyOptions(opts)
	pgxConfig, err := pgx.ParseConfig(dsn.String())
	if err != nil {
		return nil, err
	}
	pgxConfig.Logger = &logger{config.logger}
	pgxConfig.LogLevel = config.logLevel

	connStr := stdlib.RegisterConnConfig(pgxConfig)
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(config.pool.MaxOpen)
	db.SetMaxIdleConns(config.pool.MaxIdle)
	db.SetConnMaxLifetime(config.pool.MaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db, dsn}, nil
}
