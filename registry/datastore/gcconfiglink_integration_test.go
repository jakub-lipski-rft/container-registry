// +build integration

package datastore_test

import (
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadGCConfigLinkFixtures(tb testing.TB) {
	// We want to disable the trigger before loading fixtures, otherwise gc_blobs_configurations will be filled
	// by the trigger once the manifest fixtures are loaded. This will result in an error when trying to load the
	// gc_blobs_configurations fixtures, as the records already exist.
	enable, err := testutil.GCTrackConfigurationBlobsTrigger.Disable(suite.db)
	require.NoError(tb, err)
	defer enable()

	testutil.ReloadFixtures(tb, suite.db, suite.basePath,
		testutil.RepositoriesTable, testutil.BlobsTable, testutil.ManifestsTable, testutil.GCBlobsConfigurationsTable)
}

func unloadGCConfigLinkFixtures(tb testing.TB) {
	enable, err := testutil.GCTrackConfigurationBlobsTrigger.Disable(suite.db)
	require.NoError(tb, err)
	defer enable()

	require.NoError(tb, testutil.TruncateTables(suite.db,
		testutil.RepositoriesTable, testutil.BlobsTable, testutil.ManifestsTable, testutil.GCBlobsConfigurationsTable))
}

func TestGCConfigLinkStore_FindAll(t *testing.T) {
	reloadGCConfigLinkFixtures(t)

	s := datastore.NewGCConfigLinkStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/gc_blobs_configurations.sql
	expected := []*models.GCConfigLink{
		{
			ID:           1,
			RepositoryID: 3,
			ManifestID:   1,
			Digest:       "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
		},
		{
			ID:           2,
			RepositoryID: 3,
			ManifestID:   2,
			Digest:       "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
		},
		{
			ID:           3,
			RepositoryID: 4,
			ManifestID:   3,
			Digest:       "sha256:33f3ef3322b28ecfc368872e621ab715a04865471c47ca7426f3e93846157780",
		},
		{
			ID:           4,
			RepositoryID: 6,
			ManifestID:   5,
			Digest:       "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
		},
		{
			ID:           5,
			RepositoryID: 4,
			ManifestID:   9,
			Digest:       "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
		},
		{
			ID:           6,
			RepositoryID: 7,
			ManifestID:   10,
			Digest:       "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
		},
		{
			ID:           7,
			RepositoryID: 7,
			ManifestID:   11,
			Digest:       "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
		},
	}

	require.Equal(t, expected, rr)
}

func TestGCConfigLinkStore_FindAll_NotFound(t *testing.T) {
	unloadGCConfigLinkFixtures(t)

	s := datastore.NewGCConfigLinkStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestGcConfigLinkStore_Count(t *testing.T) {
	reloadGCConfigLinkFixtures(t)

	s := datastore.NewGCConfigLinkStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/gc_blobs_configurations.sql
	require.Equal(t, 7, count)
}
