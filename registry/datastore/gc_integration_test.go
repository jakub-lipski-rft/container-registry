// +build integration

package datastore_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func randomBlob(t testing.TB) *models.Blob {
	t.Helper()

	rand.Seed(time.Now().UnixNano())
	data := make([]byte, 100)
	_, err := rand.Read(data)
	require.NoError(t, err)

	return &models.Blob{
		MediaType: "application/octet-stream",
		Digest:    digest.FromBytes(data),
		Size:      rand.Int63(),
	}
}

func TestGC_TrackBlobUploads(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// create blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err := bs.Create(suite.ctx, b)
	require.NoError(t, err)

	// Check that a corresponding task was created and scheduled for 1 day ahead. This is done by the
	// `gc_track_blob_uploads` trigger/function
	brs := datastore.NewGCBlobTaskStore(suite.db)
	rr, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr))
	require.Equal(t, &models.GCBlobTask{
		ReviewAfter: b.CreatedAt.Add(24 * time.Hour),
		ReviewCount: 0,
		Digest:      b.Digest,
	}, rr[0])
}

func TestGC_TrackBlobUploads_PostponeReviewOnConflict(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// create blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err := bs.Create(suite.ctx, b)
	require.NoError(t, err)

	// delete it
	err = bs.Delete(suite.ctx, b.Digest)
	require.NoError(t, err)

	// grab existing review record (should be preserved, despite the blob deletion)
	brs := datastore.NewGCBlobTaskStore(suite.db)
	rr, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr))

	// re-create blob
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)

	// check that we still have only one review record but its due date was postponed by 1 day
	rr2, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr2))
	require.Equal(t, &models.GCBlobTask{
		ReviewAfter: b.CreatedAt.Add(24 * time.Hour),
		ReviewCount: rr[0].ReviewCount,
		Digest:      rr[0].Digest,
	}, rr2[0])
}

func TestGC_TrackBlobUploads_DoesNothingIfTriggerDisabled(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	enable, err := testutil.GCTrackBlobUploadsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()

	// create blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)

	// check that no review records were created
	brs := datastore.NewGCBlobTaskStore(suite.db)
	count, err := brs.Count(suite.ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
