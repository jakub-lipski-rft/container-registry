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
		{
			ReviewAfter: testutil.ParseTimestamp(t, "9999-12-31 23:59:59.999999", local),
			ReviewCount: 0,
			Digest:      "sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9",
		},
		{
			ReviewAfter: testutil.ParseTimestamp(t, "2020-03-03 17:57:23.405516", local),
			ReviewCount: 1,
			Digest:      "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
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
	require.Equal(t, 4, count)
}

func nextGCBlobTask(t *testing.T) (*datastore.Tx, *models.GCBlobTask) {
	t.Helper()

	tx, err := suite.db.BeginTx(suite.ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, tx)

	s := datastore.NewGCBlobTaskStore(tx)
	b, err := s.Next(suite.ctx)
	require.NoError(t, err)

	return tx, b
}

func TestGcBlobTaskStore_Next(t *testing.T) {
	// see testdata/fixtures/gc_blob_review_queue.sql
	reloadGCBlobTaskFixtures(t)

	// the 1st call should return the record with the oldest review_after
	tx1, b1 := nextGCBlobTask(t)
	defer tx1.Rollback()

	local := b1.ReviewAfter.Location()
	require.Equal(t, &models.GCBlobTask{
		ReviewAfter: testutil.ParseTimestamp(t, "2020-03-03 17:57:23.405516", local),
		ReviewCount: 1,
		Digest:      "sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073",
	}, b1)

	// The 2nd call should yield the unlocked record with the 2nd oldest review_after. In case of a draw (multiple
	// records with the same review_after), which occurs here, the returned row is the one that was first inserted.
	tx2, b2 := nextGCBlobTask(t)
	defer tx2.Rollback()

	expectedB2 := &models.GCBlobTask{
		ReviewAfter: testutil.ParseTimestamp(t, "2020-03-05 20:05:35.338639", local),
		ReviewCount: 0,
		Digest:      "sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9",
	}
	require.Equal(t, expectedB2, b2)

	// 3rd call
	tx3, b3 := nextGCBlobTask(t)
	defer tx3.Rollback()

	require.Equal(t, &models.GCBlobTask{
		ReviewAfter: testutil.ParseTimestamp(t, "2020-03-05 20:05:35.338639", local),
		ReviewCount: 3,
		Digest:      "sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21",
	}, b3)

	// Calling Next again yields nothing and does not block, as the remaining unlocked record has a review_after in
	// the future.
	tx4, b4 := nextGCBlobTask(t)
	defer tx4.Rollback()
	require.Nil(t, b4)

	// unlocking b2 and calling Next returns b2 once again
	require.NoError(t, tx2.Rollback())
	tx5, b5 := nextGCBlobTask(t)
	defer tx5.Rollback()
	require.Equal(t, expectedB2, b5)
}

func TestGcBlobTaskStore_Next_None(t *testing.T) {
	unloadGCBlobTaskFixtures(t)

	tx, b := nextGCBlobTask(t)
	defer tx.Rollback()
	require.Nil(t, b)
}
