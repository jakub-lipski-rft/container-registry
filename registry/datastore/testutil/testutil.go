package testutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/docker/distribution/registry/datastore"
	"github.com/stretchr/testify/require"
)

// table represents a table in the test database.
type table string

const (
	RepositoriesTable            table = "repositories"
	ManifestConfigurationsTable  table = "manifest_configurations"
	ManifestsTable               table = "manifests"
	RepositoryManifestsTable     table = "repository_manifests"
	BlobsTable                   table = "blobs"
	RepositoryBlobsTable         table = "repository_blobs"
	ManifestLayersTable          table = "manifest_layers"
	ManifestListsTable           table = "manifest_lists"
	ManifestListManifestsTable   table = "manifest_list_manifests"
	RepositoryManifestListsTable table = "repository_manifest_lists"
	TagsTable                    table = "tags"
)

// AllTables represents all tables in the test database.
var AllTables = []table{
	RepositoriesTable,
	ManifestConfigurationsTable,
	ManifestsTable,
	RepositoryManifestsTable,
	BlobsTable,
	RepositoryBlobsTable,
	ManifestLayersTable,
	ManifestListsTable,
	ManifestListManifestsTable,
	RepositoryManifestListsTable,
	TagsTable,
}

// truncate truncates t in the test database.
func (t table) truncate(db *datastore.DB) error {
	if _, err := db.Exec(fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", t)); err != nil {
		return fmt.Errorf("error truncating table %q: %w", t, err)
	}
	return nil
}

// seedFileName generates the expected seed filename based on the convention `<table name>.sql`.
func (t table) seedFileName() string {
	return fmt.Sprintf("%s.sql", t)
}

// DumpAsJSON dumps the table contents in JSON format using the PostgresSQL `json_agg` function. `bytea` columns are
// automatically decoded for easy visualization/comparision.
func (t table) DumpAsJSON(ctx context.Context, db datastore.Queryer) ([]byte, error) {
	var query string
	switch t {
	case ManifestConfigurationsTable:
		s := `SELECT
				json_agg(t)
			FROM (
				SELECT
					id,
					manifest_id,
					blob_id,
					created_at,
					convert_from(payload, 'UTF8')::json AS payload
				FROM %s
			) t;`
		query = fmt.Sprintf(s, t)
	case ManifestsTable:
		s := `SELECT
				json_agg(t)
			FROM (
				SELECT
					id,
					created_at,
					marked_at,
					schema_version,
					encode(digest_hex, 'hex') as digest_hex,
					convert_from(payload, 'UTF8')::json AS payload,
					media_type
				FROM %s
			) t;`
		query = fmt.Sprintf(s, t)
	case ManifestListsTable:
		s := `SELECT
				json_agg(t)
			FROM (
				SELECT
					id,
					created_at,
					marked_at,
					schema_version,
					encode(digest_hex, 'hex') as digest_hex,
					convert_from(payload, 'UTF8')::json AS payload,
					media_type
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

// NewDSN generates a new DSN for the test database based on environment variable configurations.
func NewDSN() (*datastore.DSN, error) {
	port, err := strconv.Atoi(os.Getenv("REGISTRY_DATABASE_PORT"))
	if err != nil {
		return nil, fmt.Errorf("error parsing DSN port: %w", err)
	}
	dsn := &datastore.DSN{
		Host:     os.Getenv("REGISTRY_DATABASE_HOST"),
		Port:     port,
		User:     os.Getenv("REGISTRY_DATABASE_USER"),
		Password: os.Getenv("REGISTRY_DATABASE_PASSWORD"),
		DBName:   "registry_test",
		SSLMode:  os.Getenv("REGISTRY_DATABASE_SSLMODE"),
	}

	return dsn, nil
}

// NewDB generates a new datastore.DB and opens the underlying connection.
func NewDB() (*datastore.DB, error) {
	dsn, err := NewDSN()
	if err != nil {
		return nil, err
	}

	db, err := datastore.Open(dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %w", err)
	}

	return db, nil
}

// TruncateTables truncates a set of tables in the test database.
func TruncateTables(db *datastore.DB, tables ...table) error {
	for _, table := range tables {
		if err := table.truncate(db); err != nil {
			return fmt.Errorf("error truncating tables: %w", err)
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
