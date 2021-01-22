// +build integration

package datastore_test

import (
	"testing"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadGCBlobTaskFixtures(tb testing.TB) {
	testutil.ReloadFixtures(tb, suite.db, suite.basePath, testutil.GCBlobReviewQueueTable)
}

func unloadGCBlobTaskFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(suite.db, testutil.GCBlobReviewQueueTable))
}

func TestGCBlobTaskStore_FindAll(t *testing.T) {
	reloadGCBlobTaskFixtures(t)

	s := datastore.NewGCBlobTaskStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/gc_blob_review_queue.sql
	local := rr[0].ReviewAfter.Location()
	expected := []*models.GCBlobTask{
		{
			ReviewAfter: testutil.ParseTimestamp(t, "2020-03-05 20:05:35.338639", local),
			ReviewCount: 0,
			Digest:      "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
		},
		{
			ReviewAfter: testutil.ParseTimestamp(t, "2020-03-05 20:05:35.338639", local),
			ReviewCount: 3,
			Digest:      "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
		},
	}

	require.Equal(t, expected, rr)
}

func TestGCBlobTaskStore_FindAll_NotFound(t *testing.T) {
	unloadGCBlobTaskFixtures(t)

	s := datastore.NewGCBlobTaskStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestGcBlobTaskStore_Count(t *testing.T) {
	reloadGCBlobTaskFixtures(t)

	s := datastore.NewGCBlobTaskStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/gc_blob_review_queue.sql
	require.Equal(t, 2, count)
}
