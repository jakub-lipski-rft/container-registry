package testutil

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/registry/datastore"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// table represents a table in the test database.
type table string

const (
	RepositoriesTable       table = "repositories"
	MediaTypesTable         table = "media_types"
	ManifestsTable          table = "manifests"
	ManifestReferencesTable table = "manifest_references"
	BlobsTable              table = "blobs"
	RepositoryBlobsTable    table = "repository_blobs"
	LayersTable             table = "layers"
	TagsTable               table = "tags"
)

// AllTables represents all tables in the test database.
var AllTables = []table{
	RepositoriesTable,
	ManifestsTable,
	ManifestReferencesTable,
	BlobsTable,
	RepositoryBlobsTable,
	LayersTable,
	TagsTable,
}

// truncate truncates t in the test database.
func (t table) truncate(db *datastore.DB) error {
	if _, err := db.Exec(fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", t)); err != nil {
		return fmt.Errorf("truncating table %q: %w", t, err)
	}
	return nil
}

// seedFileName generates the expected seed filename based on the convention `<table name>.sql`.
func (t table) seedFileName() string {
	return fmt.Sprintf("%s.sql", t)
}

// DumpAsJSON dumps the table contents in JSON format using the PostgresSQL `json_agg` function. `bytea` columns are
// automatically decoded for easy visualization/comparison.
func (t table) DumpAsJSON(ctx context.Context, db datastore.Queryer) ([]byte, error) {
	var query string
	switch t {
	case ManifestsTable:
		s := `SELECT
				json_agg(t)
			FROM (
				SELECT
					id,
					repository_id,
					created_at,
					schema_version,
					encode(digest, 'hex') as digest,
					convert_from(payload, 'UTF8')::json AS payload,
					media_type_id,
					configuration_media_type_id,
					convert_from(configuration_payload, 'UTF8')::json AS configuration_payload,
					encode(configuration_blob_digest, 'hex') as configuration_blob_digest
				FROM %s
			) t;`
		query = fmt.Sprintf(s, t)
	default:
		query = fmt.Sprintf("SELECT json_agg(%s) FROM %s", t, t)
	}

	var dump []byte
	row := db.QueryRowContext(ctx, query)
	if err := row.Scan(&dump); err != nil {
		return nil, err
	}

	return dump, nil
}

// NewDSNFromEnv generates a new DSN for the test database based on environment variable configurations.
func NewDSNFromEnv() (*datastore.DSN, error) {
	port, err := strconv.Atoi(os.Getenv("REGISTRY_DATABASE_PORT"))
	if err != nil {
		return nil, fmt.Errorf("parsing DSN port: %w", err)
	}
	dsn := &datastore.DSN{
		Host:        os.Getenv("REGISTRY_DATABASE_HOST"),
		Port:        port,
		User:        os.Getenv("REGISTRY_DATABASE_USER"),
		Password:    os.Getenv("REGISTRY_DATABASE_PASSWORD"),
		DBName:      "registry_test",
		SSLMode:     os.Getenv("REGISTRY_DATABASE_SSLMODE"),
		SSLCert:     os.Getenv("REGISTRY_DATABASE_SSLCERT"),
		SSLKey:      os.Getenv("REGISTRY_DATABASE_SSLKEY"),
		SSLRootCert: os.Getenv("REGISTRY_DATABASE_SSLROOTCERT"),
	}

	return dsn, nil
}

// NewDSNFromConfig generates a new DSN for the test database based on configuration options.
func NewDSNFromConfig(config configuration.Database) (*datastore.DSN, error) {
	dsn := &datastore.DSN{
		Host:        config.Host,
		Port:        config.Port,
		User:        config.User,
		Password:    config.Password,
		DBName:      "registry_test",
		SSLMode:     config.SSLMode,
		SSLCert:     config.SSLCert,
		SSLKey:      config.SSLKey,
		SSLRootCert: config.SSLRootCert,
	}

	return dsn, nil
}

func newDB(dsn *datastore.DSN, logLevel logrus.Level, logOut io.Writer) (*datastore.DB, error) {
	log := logrus.New()
	log.SetLevel(logLevel)
	log.SetOutput(logOut)

	db, err := datastore.Open(dsn, datastore.WithLogger(logrus.NewEntry(log)))
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	return db, nil
}

// NewDBFromEnv generates a new datastore.DB and opens the underlying connection based on environment variable settings.
func NewDBFromEnv() (*datastore.DB, error) {
	dsn, err := NewDSNFromEnv()
	if err != nil {
		return nil, err
	}

	logLevel, err := logrus.ParseLevel(os.Getenv("REGISTRY_LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	var logOut io.Writer
	switch os.Getenv("REGISTRY_LOG_OUTPUT") {
	case "stdout":
		logOut = os.Stdout
	case "stderr":
		logOut = os.Stderr
	case "discard":
		logOut = ioutil.Discard
	default:
		logOut = os.Stdout
	}

	return newDB(dsn, logLevel, logOut)
}

// NewDBFromConfig generates a new datastore.DB and opens the underlying connection based on configuration settings.
func NewDBFromConfig(config *configuration.Configuration) (*datastore.DB, error) {
	dsn, err := NewDSNFromConfig(config.Database)
	if err != nil {
		return nil, err
	}

	logLevel, err := logrus.ParseLevel(config.Log.Level.String())
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	var logOut io.Writer
	switch config.Log.Output {
	case configuration.LogOutputStdout:
		logOut = configuration.LogOutputStdout.Descriptor()
	case configuration.LogOutputStderr:
		logOut = configuration.LogOutputStderr.Descriptor()
	case configuration.LogOutputDiscard:
	default:
		logOut = configuration.LogOutputStdout.Descriptor()
	}

	return newDB(dsn, logLevel, logOut)
}

// TruncateTables truncates a set of tables in the test database.
func TruncateTables(db *datastore.DB, tables ...table) error {
	for _, table := range tables {
		if err := table.truncate(db); err != nil {
			return fmt.Errorf("truncating tables: %w", err)
		}
	}
	return nil
}

// TruncateAllTables truncates all tables in the test database.
func TruncateAllTables(db *datastore.DB) error {
	return TruncateTables(db, AllTables...)
}

// ReloadFixtures truncates all a given set of tables and then injects related fixtures.
func ReloadFixtures(tb testing.TB, db *datastore.DB, basePath string, tables ...table) {
	tb.Helper()

	require.NoError(tb, TruncateTables(db, tables...))

	for _, table := range tables {
		path := filepath.Join(basePath, "testdata", "fixtures", table.seedFileName())

		query, err := ioutil.ReadFile(path)
		require.NoErrorf(tb, err, "error reading fixture")

		_, err = db.Exec(string(query))
		require.NoErrorf(tb, err, "error loading fixture")
	}
}

// ParseTimestamp parses a timestamp into a time.Time, matching a given location.
func ParseTimestamp(tb testing.TB, timestamp string, location *time.Location) time.Time {
	tb.Helper()

	t, err := time.Parse("2006-01-02 15:04:05.000000", timestamp)
	require.NoError(tb, err)

	return t.In(location)
}

func createGoldenFile(tb testing.TB, path string) {
	tb.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		tb.Log("creating .golden file")

		f, err := os.Create(path)
		require.NoError(tb, err, "error creating .golden file")
		require.NoError(tb, f.Close())
	}
}

func updateGoldenFile(tb testing.TB, path string, content []byte) {
	tb.Helper()

	tb.Log("updating .golden file")
	err := ioutil.WriteFile(path, content, 0644)
	require.NoError(tb, err, "error updating .golden file")
}

func readGoldenFile(tb testing.TB, path string) []byte {
	tb.Helper()

	content, err := ioutil.ReadFile(path)
	require.NoError(tb, err, "error reading .golden file")

	return content
}

// CompareWithGoldenFile compares an actual value with the content of a .golden file. If requested, a missing golden
// file is automatically created and an outdated golden file automatically updated to match the actual content.
func CompareWithGoldenFile(tb testing.TB, path string, actual []byte, create, update bool) {
	tb.Helper()

	if create {
		createGoldenFile(tb, path)
	}
	if update {
		updateGoldenFile(tb, path, actual)
	}

	expected := readGoldenFile(tb, path)
	require.Equal(tb, string(expected), string(actual), "does not match .golden file")
}
