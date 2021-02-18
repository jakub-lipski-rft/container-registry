// +build integration

package datastore_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/docker/distribution/registry/datastore"
	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadGCManifestTaskFixtures(tb testing.TB) {
	reloadManifestFixtures(tb)
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
			RepositoryID: 4,
			ManifestID:   7,
			ReviewAfter:  testutil.ParseTimestamp(t, "2020-04-03 18:45:04.470711", local),
			ReviewCount:  2,
		},
		{
			RepositoryID: 4,
			ManifestID:   9,
			ReviewAfter:  testutil.ParseTimestamp(t, "9999-12-31 23:59:59.999999", local),
			ReviewCount:  0,
		},
		{
			RepositoryID: 3,
			ManifestID:   1,
			ReviewAfter:  testutil.ParseTimestamp(t, "2020-03-03 17:50:26.461745", local),
			ReviewCount:  0,
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
	require.Equal(t, 3, count)
}

func nextGCManifestTask(t *testing.T) (datastore.Transactor, *models.GCManifestTask) {
	t.Helper()

	tx, err := suite.db.BeginTx(suite.ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, tx)

	s := datastore.NewGCManifestTaskStore(tx)
	m, err := s.Next(suite.ctx)
	require.NoError(t, err)

	return tx, m
}

func TestGcManifestTaskStore_Next(t *testing.T) {
	// see testdata/fixtures/gc_manifest_review_queue.sql
	reloadGCManifestTaskFixtures(t)

	// the 1st call should return the record with the oldest review_after
	tx1, m1 := nextGCManifestTask(t)
	defer tx1.Rollback()

	local := m1.ReviewAfter.Location()
	require.Equal(t, &models.GCManifestTask{
		RepositoryID: 3,
		ManifestID:   1,
		ReviewAfter:  testutil.ParseTimestamp(t, "2020-03-03 17:50:26.461745", local),
		ReviewCount:  0,
	}, m1)

	// The 2nd call should yield the unlocked record with the 2nd oldest review_after. In case of a draw (multiple
	// records with the same review_after), which occurs here, the returned row is the one that was first inserted.
	tx2, m2 := nextGCManifestTask(t)
	defer tx2.Rollback()

	expectedM2 := &models.GCManifestTask{
		RepositoryID: 4,
		ManifestID:   7,
		ReviewAfter:  testutil.ParseTimestamp(t, "2020-04-03 18:45:04.470711", local),
		ReviewCount:  2,
	}
	require.Equal(t, expectedM2, m2)

	// Calling Next again yields nothing and does not block, as the remaining unlocked record has a review_after in
	// the future.
	tx3, m3 := nextGCManifestTask(t)
	defer tx3.Rollback()
	require.Nil(t, m3)

	// unlocking m2 and calling Next returns m2 once again
	require.NoError(t, tx2.Rollback())
	tx4, m4 := nextGCManifestTask(t)
	defer tx4.Rollback()
	require.Equal(t, expectedM2, m4)
}

func TestGcManifestTaskStore_Next_None(t *testing.T) {
	unloadGCManifestTaskFixtures(t)

	tx, m := nextGCManifestTask(t)
	defer tx.Rollback()
	require.Nil(t, m)
}

func TestGcManifestTaskStore_Postpone(t *testing.T) {
	// see testdata/fixtures/gc_manifest_review_queue.sql
	reloadGCManifestTaskFixtures(t)

	tx, m := nextGCManifestTask(t)
	defer tx.Rollback()

	oldReviewAfter := m.ReviewAfter
	oldReviewCount := m.ReviewCount
	d := 24 * time.Hour

	s := datastore.NewGCManifestTaskStore(tx)
	err := s.Postpone(suite.ctx, m, d)
	require.NoError(t, err)
	require.Equal(t, oldReviewAfter.Add(d), m.ReviewAfter)
	require.Equal(t, oldReviewCount+1, m.ReviewCount)
}

func TestGcManifestTaskStore_Postpone_NotFound(t *testing.T) {
	tx, err := suite.db.BeginTx(suite.ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	s := datastore.NewGCManifestTaskStore(tx)
	err = s.Postpone(suite.ctx, &models.GCManifestTask{RepositoryID: 1, ManifestID: 3}, 0)
	require.EqualError(t, err, "GC manifest task not found")
}

func existsGCManifestTask(t *testing.T, db datastore.Queryer, repositoryID, manifestID int64) bool {
	t.Helper()

	q := `SELECT
			EXISTS (
				SELECT
					1
				FROM
					gc_manifest_review_queue
				WHERE
					repository_id = $1
					AND manifest_id = $2)`

	var exists bool
	require.NoError(t, db.QueryRowContext(suite.ctx, q, repositoryID, manifestID).Scan(&exists))

	return exists
}

func TestExistsGCManifestTask(t *testing.T) {
	// see testdata/fixtures/gc_manifest_review_queue.sql
	reloadGCManifestTaskFixtures(t)

	require.True(t, existsGCManifestTask(t, suite.db, 4, 7))
	require.False(t, existsGCManifestTask(t, suite.db, 6, 2))
}

func pickGCManifestTask(t *testing.T, db datastore.Queryer, repositoryID, manifestID int64) *models.GCManifestTask {
	t.Helper()

	q := `SELECT
			review_after,
			review_count
		FROM
			gc_manifest_review_queue
		WHERE
			repository_id = $1
			AND manifest_id = $2
		FOR UPDATE`

	m := &models.GCManifestTask{RepositoryID: repositoryID, ManifestID: manifestID}
	if err := db.QueryRowContext(suite.ctx, q, m.RepositoryID, m.ManifestID).Scan(&m.ReviewAfter, &m.ReviewCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		t.Error(err)
	}

	return m
}

func TestPickGCManifestTask(t *testing.T) {
	// see testdata/fixtures/gc_manifest_review_queue.sql
	reloadGCManifestTaskFixtures(t)

	tx, err := suite.db.BeginTx(suite.ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	repositoryID := int64(4)
	manifestID := int64(7)
	m := pickGCManifestTask(t, tx, repositoryID, manifestID)

	require.Equal(t, &models.GCManifestTask{
		RepositoryID: repositoryID,
		ManifestID:   manifestID,
		ReviewAfter:  testutil.ParseTimestamp(t, "2020-04-03 18:45:04.470711", m.ReviewAfter.Location()),
		ReviewCount:  2,
	}, m)
	require.Nil(t, pickGCManifestTask(t, tx, 6, 2))
}

func TestGcManifestTaskStore_Delete(t *testing.T) {
	// see testdata/fixtures/gc_manifest_review_queue.sql
	reloadGCManifestTaskFixtures(t)

	tx, m := nextGCManifestTask(t)
	defer tx.Rollback()

	s := datastore.NewGCManifestTaskStore(tx)
	err := s.Delete(suite.ctx, m)
	require.NoError(t, err)
	require.False(t, existsGCManifestTask(t, tx, m.RepositoryID, m.ManifestID))
}

func TestGcManifestTaskStore_Delete_NotFound(t *testing.T) {
	tx, err := suite.db.BeginTx(suite.ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	s := datastore.NewGCManifestTaskStore(tx)
	err = s.Delete(suite.ctx, &models.GCManifestTask{RepositoryID: 6, ManifestID: 2})
	require.EqualError(t, err, "GC manifest task not found")
}

func TestGcManifestTaskStore_IsDangling_Yes(t *testing.T) {
	s := datastore.NewGCManifestTaskStore(suite.db)
	yn, err := s.IsDangling(suite.ctx, &models.GCManifestTask{RepositoryID: 6, ManifestID: 2})
	require.NoError(t, err)
	require.True(t, yn)
}

func TestGcManifestTaskStore_IsDangling_No_Tagged(t *testing.T) {
	// see testdata/fixtures/[gc_manifest_review_queue|tags].sql
	reloadGCManifestTaskFixtures(t)

	tx, err := suite.db.BeginTx(suite.ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	m := pickGCManifestTask(t, tx, 4, 7)
	require.NotNil(t, m)

	s := datastore.NewGCManifestTaskStore(tx)
	yn, err := s.IsDangling(suite.ctx, m)
	require.NoError(t, err)
	require.False(t, yn)
}

func TestGcManifestTaskStore_IsDangling_No_ReferencedByList(t *testing.T) {
	// see testdata/fixtures/[gc_manifest_review_queue|manifest_references].sql
	reloadGCManifestTaskFixtures(t)

	tx, err := suite.db.BeginTx(suite.ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	m := pickGCManifestTask(t, tx, 4, 9)
	require.NotNil(t, m)

	s := datastore.NewGCManifestTaskStore(tx)
	yn, err := s.IsDangling(suite.ctx, m)
	require.NoError(t, err)
	require.False(t, yn)
}

func TestGcManifestTaskStore_IsDangling_No_TaggedAndReferencedByList(t *testing.T) {
	// see testdata/fixtures/[gc_manifest_review_queue|tags|manifest_references].sql
	reloadGCManifestTaskFixtures(t)

	tx, err := suite.db.BeginTx(suite.ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	// tag an (untagged) manifest referenced by a manifest list
	ts := datastore.NewTagStore(tx)
	err = ts.CreateOrUpdate(suite.ctx, &models.Tag{RepositoryID: 4, ManifestID: 9, Name: "foo"})
	require.NoError(t, err)

	m := pickGCManifestTask(t, tx, 4, 9)
	require.NotNil(t, m)

	s := datastore.NewGCManifestTaskStore(tx)
	yn, err := s.IsDangling(suite.ctx, m)
	require.NoError(t, err)
	require.False(t, yn)
}
