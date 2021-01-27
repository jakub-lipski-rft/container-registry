// +build integration

package datastore_test

import (
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadGCManifestTaskFixtures(tb testing.TB) {
	testutil.ReloadFixtures(tb, suite.db, suite.basePath, testutil.GCManifestReviewQueueTable)
}

func unloadGCManifestTaskFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(suite.db, testutil.GCManifestReviewQueueTable))
}

func TestGCManifestTaskStore_FindAll(t *testing.T) {
	reloadGCManifestTaskFixtures(t)

	s := datastore.NewGCManifestTaskStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/gc_manifest_review_queue.sql
	local := rr[0].ReviewAfter.Location()
	expected := []*models.GCManifestTask{
		{
			RepositoryID: 3,
			ManifestID:   1,
			ReviewAfter:  testutil.ParseTimestamp(t, "2020-03-03 17:50:26.461745", local),
			ReviewCount:  0,
		},
		{
			RepositoryID: 4,
			ManifestID:   7,
			ReviewAfter:  testutil.ParseTimestamp(t, "2020-04-03 18:45:04.470711", local),
			ReviewCount:  2,
		},
	}

	require.Equal(t, expected, rr)
}

func TestGCManifestTaskStore_FindAll_NotFound(t *testing.T) {
	unloadGCManifestTaskFixtures(t)

	s := datastore.NewGCManifestTaskStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestGcManifestTaskStore_Count(t *testing.T) {
	reloadGCManifestTaskFixtures(t)

	s := datastore.NewGCManifestTaskStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/gc_manifest_review_queue.sql
	require.Equal(t, 2, count)
}
