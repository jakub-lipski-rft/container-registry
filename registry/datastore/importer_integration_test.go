// +build integration

package datastore_test

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/docker/distribution"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/docker/distribution/registry/storage"
	storageDriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/filesystem"
	"github.com/docker/libtrust"
	"github.com/stretchr/testify/require"
)

func newFilesystemStorageDriver(tb testing.TB) *filesystem.Driver {
	tb.Helper()

	driver, err := filesystem.FromParameters(map[string]interface{}{
		"rootdirectory": path.Join(suite.fixturesPath, "importer"),
	})
	require.NoError(tb, err, "error creating storage driver")

	return driver
}

func newRegistry(tb testing.TB, driver storageDriver.StorageDriver) distribution.Namespace {
	tb.Helper()

	// load custom key to be used for manifest signing, ensuring that we have reproducible signatures
	pemKey, err := ioutil.ReadFile(path.Join(suite.fixturesPath, "keys", "manifest_sign"))
	require.NoError(tb, err)
	block, _ := pem.Decode([]byte(pemKey))
	require.NotNil(tb, block)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(tb, err)
	k, err := libtrust.FromCryptoPrivateKey(privateKey)
	require.NoErrorf(tb, err, "error loading signature key")

	registry, err := storage.NewRegistry(suite.ctx, driver, storage.Schema1SigningKey(k), storage.EnableSchema1)
	require.NoError(tb, err, "error creating registry")

	return registry
}

// overrideDynamicData is required to override all attributes that change with every test run. This is needed to ensure
// that we can consistently compare the output of a database dump with the stored reference snapshots (.golden files).
// For example, all entities have a `created_at` attribute that changes with every test run, therefore, when we dump
// the database we have to override this attribute value so that it matches the one in the .golden files.
func overrideDynamicData(tb testing.TB, actual []byte) []byte {
	tb.Helper()

	// the created_at timestamps for all entities change with every test run
	re := regexp.MustCompile(`"created_at":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d*\+\d{2}:\d{2}"`)
	actual = re.ReplaceAllLiteral(actual, []byte(`"created_at":"2020-04-15T12:04:28.95584"`))

	// schema 1 manifests have `signature` and `protected` attributes that changes with every test run
	re = regexp.MustCompile(`"signature": ".*"`)
	actual = re.ReplaceAllLiteral(actual, []byte(`"signature": "lBzn6_e7f0mdqQXKhkRMdI"`))
	re = regexp.MustCompile(`"protected": ".*"`)
	actual = re.ReplaceAllLiteral(actual, []byte(`"protected": "eyJmb3JtYXRMZW5ndGgiOj"`))

	return actual
}

func newImporter(t *testing.T, db *datastore.DB, opts ...datastore.ImporterOption) *datastore.Importer {
	t.Helper()

	driver := newFilesystemStorageDriver(t)
	registry := newRegistry(t, driver)

	return datastore.NewImporter(db, driver, registry, opts...)
}

// Dump each table as JSON and compare the output against reference snapshots (.golden files)
func validateImport(t *testing.T, db *datastore.DB) {
	t.Helper()

	for _, tt := range testutil.AllTables {
		t.Run(string(tt), func(t *testing.T) {
			actual, err := tt.DumpAsJSON(suite.ctx, db)
			require.NoError(t, err, "error dumping table")

			// see testdata/golden/<test name>/<table>.golden
			p := filepath.Join(suite.goldenPath, t.Name()+".golden")
			actual = overrideDynamicData(t, actual)
			testutil.CompareWithGoldenFile(t, p, actual, *create, *update)
		})
	}
}

func TestImporter_ImportAll(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	imp := newImporter(t, suite.db, datastore.WithImportDanglingManifests)
	require.NoError(t, imp.ImportAll(suite.ctx))
	validateImport(t, suite.db)
}

func TestImporter_ImportAll_DanglingBlobs(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	imp := newImporter(t, suite.db, datastore.WithImportDanglingManifests, datastore.WithImportDanglingBlobs)
	require.NoError(t, imp.ImportAll(suite.ctx))
	validateImport(t, suite.db)
}

func TestImporter_ImportAll_AllowIdempotent(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// First, import a single repository, only tagged manifests and referenced blobs.
	imp1 := newImporter(t, suite.db)
	require.NoError(t, imp1.Import(suite.ctx, "f-dangling-manifests"))

	// Now try to import the entire contents of the registry including what was previously imported.
	imp2 := newImporter(t, suite.db, datastore.WithImportDanglingManifests, datastore.WithImportDanglingBlobs)
	require.NoError(t, imp2.ImportAll(suite.ctx))
	validateImport(t, suite.db)
}

func TestImporter_ImportAll_DryRun(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	imp := newImporter(t, suite.db, datastore.WithDryRun)
	require.NoError(t, imp.ImportAll(suite.ctx))
	validateImport(t, suite.db)
}

func TestImporter_ImportAll_DryRunDanglingBlobs(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	imp := newImporter(t, suite.db, datastore.WithDryRun, datastore.WithImportDanglingBlobs)
	require.NoError(t, imp.ImportAll(suite.ctx))
	validateImport(t, suite.db)
}

func TestImporter_ImportAll_AbortsIfDatabaseIsNotEmpty(t *testing.T) {
	driver := newFilesystemStorageDriver(t)
	registry := newRegistry(t, driver)

	// load some fixtures
	reloadRepositoryFixtures(t)

	imp := datastore.NewImporter(suite.db, driver, registry, datastore.WithImportDanglingManifests, datastore.WithRequireEmptyDatabase)
	err := imp.ImportAll(suite.ctx)
	require.EqualError(t, err, "non-empty database")
}

func TestImporter_Import(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	imp := newImporter(t, suite.db, datastore.WithImportDanglingManifests)
	require.NoError(t, imp.Import(suite.ctx, "b-nested/older"))
	validateImport(t, suite.db)
}

func TestImporter_Import_TaggedOnly(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	imp := newImporter(t, suite.db)
	require.NoError(t, imp.Import(suite.ctx, "f-dangling-manifests"))
	validateImport(t, suite.db)
}

func TestImporter_Import_AllowIdempotent(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// Import a single repository twice, should succeed.
	imp := newImporter(t, suite.db, datastore.WithImportDanglingManifests)
	require.NoError(t, imp.Import(suite.ctx, "b-nested/older"))
	require.NoError(t, imp.Import(suite.ctx, "b-nested/older"))
	validateImport(t, suite.db)
}

func TestImporter_Import_DryRun(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	imp := newImporter(t, suite.db, datastore.WithDryRun)
	require.NoError(t, imp.Import(suite.ctx, "a-simple"))
	validateImport(t, suite.db)
}

func TestImporter_Import_AbortsIfDatabaseIsNotEmpty(t *testing.T) {
	driver := newFilesystemStorageDriver(t)
	registry := newRegistry(t, driver)

	// load some fixtures
	reloadRepositoryFixtures(t)

	imp := datastore.NewImporter(suite.db, driver, registry, datastore.WithImportDanglingManifests, datastore.WithRequireEmptyDatabase)
	err := imp.Import(suite.ctx, "a-simple")
	require.EqualError(t, err, "non-empty database")
}
