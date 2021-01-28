// +build integration

package datastore_test

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func randomDigest(t testing.TB) digest.Digest {
	t.Helper()

	rand.Seed(time.Now().UnixNano())
	data := make([]byte, 100)
	_, err := rand.Read(data)
	require.NoError(t, err)

	return digest.FromBytes(data)
}

func randomBlob(t testing.TB) *models.Blob {
	t.Helper()

	rand.Seed(time.Now().UnixNano())
	return &models.Blob{
		MediaType: "application/octet-stream",
		Digest:    randomDigest(t),
		Size:      rand.Int63(),
	}
}

func randomRepository(t testing.TB) *models.Repository {
	t.Helper()

	rand.Seed(time.Now().UnixNano())
	n := strconv.Itoa(rand.Int())
	return &models.Repository{
		Name: n,
		Path: n,
	}
}

func randomManifest(t testing.TB, r *models.Repository, configBlob *models.Blob) *models.Manifest {
	t.Helper()

	m := &models.Manifest{
		RepositoryID:  r.ID,
		SchemaVersion: 2,
		MediaType:     schema2.MediaTypeManifest,
		Digest:        randomDigest(t),
		Payload:       models.Payload(`{"foo": "bar"}`),
	}
	if configBlob != nil {
		m.Configuration = &models.Configuration{
			MediaType: schema2.MediaTypeImageConfig,
			Digest:    configBlob.Digest,
			Payload:   models.Payload(`{"foo": "bar"}`),
		}
	}

	return m
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

	// check that we still have only one review record but its due date was postponed to now (re-create time) + 1 day
	rr2, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr2))
	require.Equal(t, rr[0].ReviewCount, rr2[0].ReviewCount)
	require.Equal(t, rr[0].Digest, rr2[0].Digest)
	// this is fast, so review_after is only a few milliseconds ahead of the original time
	require.True(t, rr2[0].ReviewAfter.After(rr[0].ReviewAfter))
	require.WithinDuration(t, rr[0].ReviewAfter, rr2[0].ReviewAfter, 100*time.Millisecond)
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

func TestGC_TrackConfigurationBlobs(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err := rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create config blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, b)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// Check that a corresponding task was created and scheduled for 1 day ahead. This is done by the
	// `gc_track_configuration_blobs` trigger/function
	brs := datastore.NewGCConfigLinkStore(suite.db)
	rr, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr))
	require.NotEmpty(t, rr[0].ID)
	require.Equal(t, r.ID, rr[0].RepositoryID)
	require.Equal(t, m.ID, rr[0].ManifestID)
	require.Equal(t, b.Digest, rr[0].Digest)
}

func TestGC_TrackConfigurationBlobs_DoesNothingIfTriggerDisabled(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	enable, err := testutil.GCTrackConfigurationBlobsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err = rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create config blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, b)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// check that no records were created
	brs := datastore.NewGCConfigLinkStore(suite.db)
	count, err := brs.Count(suite.ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestGC_TrackLayerBlobs(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err := rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create layer blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, nil)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// associate layer with manifest
	err = ms.AssociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	// Check that a corresponding row was created. This is done by the `gc_track_layer_blobs` trigger/function
	brs := datastore.NewGCLayerLinkStore(suite.db)
	ll, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(ll))
	require.NotEmpty(t, ll[0].ID)
	require.Equal(t, r.ID, ll[0].RepositoryID)
	require.Equal(t, int64(1), ll[0].LayerID)
	require.Equal(t, b.Digest, ll[0].Digest)
}

func TestGC_TrackLayerBlobs_DoesNothingIfTriggerDisabled(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	enable, err := testutil.GCTrackLayerBlobsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err = rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create layer blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, nil)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// associate layer with manifest
	err = ms.AssociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	// check that no records were created
	brs := datastore.NewGCConfigLinkStore(suite.db)
	count, err := brs.Count(suite.ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestGC_TrackManifestUploads(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// create repository
	rs := datastore.NewRepositoryStore(suite.db)
	r := randomRepository(t)
	err := rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, nil)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// Check that a corresponding task was created and scheduled for 1 day ahead. This is done by the
	// `gc_track_manifest_uploads` trigger/function
	brs := datastore.NewGCManifestTaskStore(suite.db)
	tt, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(tt))
	require.Equal(t, &models.GCManifestTask{
		RepositoryID: r.ID,
		ManifestID:   m.ID,
		ReviewAfter:  m.CreatedAt.Add(24 * time.Hour),
		ReviewCount:  0,
	}, tt[0])
}

func TestGC_TrackManifestUploads_DoesNothingIfTriggerDisabled(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	enable, err := testutil.GCTrackManifestUploadsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()

	// create repository
	rs := datastore.NewRepositoryStore(suite.db)
	r := randomRepository(t)
	err = rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, nil)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// check that no review records were created
	mrs := datastore.NewGCManifestTaskStore(suite.db)
	count, err := mrs.Count(suite.ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestGC_TrackDeletedManifests(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// disable other triggers that also insert on gc_blob_review_queue so that they don't interfere with this test
	enable, err := testutil.GCTrackConfigurationBlobsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()
	enable, err = testutil.GCTrackBlobUploadsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err = rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create config blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)
	err = rs.LinkBlob(suite.ctx, r, b.Digest)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, b)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// confirm that the review queue remains empty
	brs := datastore.NewGCBlobTaskStore(suite.db)
	count, err := brs.Count(suite.ctx)
	require.NoError(t, err)
	require.Zero(t, count)

	// delete manifest
	ok, err := rs.DeleteManifest(suite.ctx, r, m.Digest)
	require.NoError(t, err)
	require.True(t, ok)

	// check that a corresponding task was created for the config blob and scheduled for 1 day ahead
	tt, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(tt))
	require.Equal(t, 0, tt[0].ReviewCount)
	require.Equal(t, b.Digest, tt[0].Digest)
	// ignore the few milliseconds between blob creation and queueing for review in response to the manifest delete
	require.WithinDuration(t, tt[0].ReviewAfter, b.CreatedAt.Add(24*time.Hour), 100*time.Millisecond)
}

func TestGC_TrackDeletedManifests_PostponeReviewOnConflict(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err := rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create config blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)
	err = rs.LinkBlob(suite.ctx, r, b.Digest)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, b)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// grab existing review record (created by the gc_track_blob_uploads_trigger trigger)
	brs := datastore.NewGCBlobTaskStore(suite.db)
	rr, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr))

	// delete manifest
	ok, err := rs.DeleteManifest(suite.ctx, r, m.Digest)
	require.NoError(t, err)
	require.True(t, ok)

	// check that we still have only one review record but its due date was postponed to now (delete time) + 1 day
	rr2, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr2))
	require.Equal(t, rr[0].ReviewCount, rr2[0].ReviewCount)
	require.Equal(t, rr[0].Digest, rr2[0].Digest)
	// this is fast, so review_after is only a few milliseconds ahead of the original time
	require.True(t, rr2[0].ReviewAfter.After(rr[0].ReviewAfter))
	require.LessOrEqual(t, rr2[0].ReviewAfter.Sub(rr[0].ReviewAfter).Milliseconds(), int64(100))
}

func TestGC_TrackDeletedManifests_DoesNothingIfTriggerDisabled(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	enable, err := testutil.GCTrackDeletedManifestsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()
	// disable other triggers that also insert on gc_blob_review_queue so that they don't interfere with this test
	enable, err = testutil.GCTrackConfigurationBlobsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()
	enable, err = testutil.GCTrackBlobUploadsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err = rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create config blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)
	err = rs.LinkBlob(suite.ctx, r, b.Digest)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, b)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// delete manifest
	ok, err := rs.DeleteManifest(suite.ctx, r, m.Digest)
	require.NoError(t, err)
	require.True(t, ok)

	// check that no review records were created
	brs := datastore.NewGCBlobTaskStore(suite.db)
	count, err := brs.Count(suite.ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestGC_TrackDeletedLayers(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// disable other triggers that also insert on gc_blob_review_queue so that they don't interfere with this test
	enable, err := testutil.GCTrackBlobUploadsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err = rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create layer blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)
	err = rs.LinkBlob(suite.ctx, r, b.Digest)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, nil)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// associate layer with manifest
	err = ms.AssociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	// confirm that the review queue remains empty
	brs := datastore.NewGCBlobTaskStore(suite.db)
	count, err := brs.Count(suite.ctx)
	require.NoError(t, err)
	require.Zero(t, count)

	// dissociate layer blob
	err = ms.DissociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	// check that a corresponding task was created for the layer blob and scheduled for 1 day ahead
	tt, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(tt))
	require.Equal(t, 0, tt[0].ReviewCount)
	require.Equal(t, b.Digest, tt[0].Digest)
	// ignore the few milliseconds between blob creation and queueing for review in response to the layer dissociation
	require.WithinDuration(t, tt[0].ReviewAfter, b.CreatedAt.Add(24*time.Hour), 100*time.Millisecond)
}

func TestGC_TrackDeletedLayers_PostponeReviewOnConflict(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err := rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create layer blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)
	err = rs.LinkBlob(suite.ctx, r, b.Digest)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, nil)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// associate layer with manifest
	err = ms.AssociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	// grab existing review record (created by the gc_track_blob_uploads_trigger trigger)
	brs := datastore.NewGCBlobTaskStore(suite.db)
	rr, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr))

	// dissociate layer blob
	err = ms.DissociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	// check that we still have only one review record but its due date was postponed to now (delete time) + 1 day
	rr2, err := brs.FindAll(suite.ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(rr2))
	require.Equal(t, rr[0].ReviewCount, rr2[0].ReviewCount)
	require.Equal(t, rr[0].Digest, rr2[0].Digest)
	// this is fast, so review_after is only a few milliseconds ahead of the original time
	require.True(t, rr2[0].ReviewAfter.After(rr[0].ReviewAfter))
	require.LessOrEqual(t, rr2[0].ReviewAfter.Sub(rr[0].ReviewAfter).Milliseconds(), int64(100))
}

func TestGC_TrackDeletedLayers_DoesNothingIfTriggerDisabled(t *testing.T) {
	require.NoError(t, testutil.TruncateAllTables(suite.db))

	enable, err := testutil.GCTrackDeletedLayersTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()
	// disable other triggers that also insert on gc_blob_review_queue so that they don't interfere with this test
	enable, err = testutil.GCTrackBlobUploadsTrigger.Disable(suite.db)
	require.NoError(t, err)
	defer enable()

	// create repo
	r := randomRepository(t)
	rs := datastore.NewRepositoryStore(suite.db)
	err = rs.Create(suite.ctx, r)
	require.NoError(t, err)

	// create layer blob
	bs := datastore.NewBlobStore(suite.db)
	b := randomBlob(t)
	err = bs.Create(suite.ctx, b)
	require.NoError(t, err)
	err = rs.LinkBlob(suite.ctx, r, b.Digest)
	require.NoError(t, err)

	// create manifest
	ms := datastore.NewManifestStore(suite.db)
	m := randomManifest(t, r, nil)
	err = ms.Create(suite.ctx, m)
	require.NoError(t, err)

	// associate layer with manifest
	err = ms.AssociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	// dissociate layer blob
	err = ms.DissociateLayerBlob(suite.ctx, m, b)
	require.NoError(t, err)

	// check that no review records were created
	brs := datastore.NewGCBlobTaskStore(suite.db)
	count, err := brs.Count(suite.ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
