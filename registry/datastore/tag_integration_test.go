// +build integration

package datastore_test

import (
	"database/sql"
	"testing"

	"github.com/docker/distribution/registry/datastore"

	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadTagFixtures(tb testing.TB) {
	testutil.ReloadFixtures(
		tb, suite.db, suite.basePath,
		// A Tag has a foreign key for a Manifest, which in turn references a Repository (insert order matters)
		testutil.RepositoriesTable, testutil.ManifestsTable, testutil.TagsTable,
	)
}

func unloadTagFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(
		suite.db,
		// A Tag has a foreign key for a Manifest, which in turn references a Repository (insert order matters)
		testutil.RepositoriesTable, testutil.ManifestsTable, testutil.TagsTable,
	))
}

func TestTagStore_FindByID(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	tag, err := s.FindByID(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	expected := &models.Tag{
		ID:         1,
		Name:       "1.0.0",
		ManifestID: 1,
		CreatedAt:  testutil.ParseTimestamp(t, "2020-03-02 17:57:43.283783", tag.CreatedAt.Location()),
	}
	require.Equal(t, expected, tag)
}

func TestTagStore_FindByID_NotFound(t *testing.T) {
	s := datastore.NewTagStore(suite.db)

	r, err := s.FindByID(suite.ctx, 0)
	require.Nil(t, r)
	require.EqualError(t, err, "tag not found")
}

func TestTagStore_FindByNameAndManifestID(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	tag, err := s.FindByNameAndManifestID(suite.ctx, "2.0.0", 2)
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	excepted := &models.Tag{
		ID:         2,
		Name:       "2.0.0",
		ManifestID: 2,
		CreatedAt:  testutil.ParseTimestamp(t, "2020-03-02 17:57:44.283783", tag.CreatedAt.Location()),
	}
	require.Equal(t, excepted, tag)
}

func TestTagStore_FindByNameAndManifestID_NotFound(t *testing.T) {
	s := datastore.NewTagStore(suite.db)

	tag, err := s.FindByNameAndManifestID(suite.ctx, "3.0.0", 2)
	require.Nil(t, tag)
	require.EqualError(t, err, "tag not found")
}

func TestTagStore_FindAll(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	tt, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	local := tt[0].CreatedAt.Location()
	expected := models.Tags{
		{
			ID:         1,
			Name:       "1.0.0",
			ManifestID: 1,
			CreatedAt:  testutil.ParseTimestamp(t, "2020-03-02 17:57:43.283783", local),
		},
		{
			ID:         2,
			Name:       "2.0.0",
			ManifestID: 2,
			CreatedAt:  testutil.ParseTimestamp(t, "2020-03-02 17:57:44.283783", local),
		},
		{
			ID:         3,
			Name:       "latest",
			ManifestID: 2,
			CreatedAt:  testutil.ParseTimestamp(t, "2020-03-02 17:57:45.283783", local),
			UpdatedAt: sql.NullTime{
				Time:  testutil.ParseTimestamp(t, "2020-03-02 17:57:53.029514", local),
				Valid: true,
			},
		},
		{
			ID:         4,
			Name:       "1.0.0",
			ManifestID: 3,
			CreatedAt:  testutil.ParseTimestamp(t, "2020-03-02 17:57:46.283783", local),
		},
		{
			ID:         5,
			Name:       "latest",
			ManifestID: 3,
			CreatedAt:  testutil.ParseTimestamp(t, "2020-03-02 17:57:47.283783", local),
		},
	}
	require.Equal(t, expected, tt)
}

func TestTagStore_FindAll_NotFound(t *testing.T) {
	unloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	tt, err := s.FindAll(suite.ctx)
	require.Empty(t, tt)
	require.NoError(t, err)
}

func TestTagStore_Count(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	require.Equal(t, 5, count)
}

func TestTagStore_Create(t *testing.T) {
	unloadTagFixtures(t)
	reloadRepositoryFixtures(t)
	reloadManifestFixtures(t)

	s := datastore.NewTagStore(suite.db)
	tag := &models.Tag{
		Name:       "3.0.0",
		ManifestID: 1,
	}
	err := s.Create(suite.ctx, tag)

	require.NoError(t, err)
	require.NotEmpty(t, tag.ID)
	require.NotEmpty(t, tag.CreatedAt)
}

func TestTagStore_Create_DuplicateFails(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)
	tag := &models.Tag{
		Name:       "1.0.0",
		ManifestID: 1,
	}
	err := s.Create(suite.ctx, tag)
	require.Error(t, err)
}

func TestTagStore_Update(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)
	update := &models.Tag{
		ID:         5,
		Name:       "2.0.0",
		ManifestID: 1,
	}
	err := s.Update(suite.ctx, update)
	require.NoError(t, err)

	r, err := s.FindByID(suite.ctx, update.ID)
	require.NoError(t, err)

	update.CreatedAt = r.CreatedAt
	require.Equal(t, update, r)
}

func TestTagStore_Update_NotFound(t *testing.T) {
	s := datastore.NewTagStore(suite.db)

	update := &models.Tag{
		ID:         100,
		Name:       "foo",
		ManifestID: 1,
	}

	err := s.Update(suite.ctx, update)
	require.EqualError(t, err, "tag not found")
}

func TestTagStore_SoftDelete(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	r := &models.Tag{ID: 1}
	err := s.SoftDelete(suite.ctx, r)
	require.NoError(t, err)

	r, err = s.FindByID(suite.ctx, r.ID)
	require.NoError(t, err)

	require.True(t, r.DeletedAt.Valid)
	require.NotEmpty(t, r.DeletedAt.Time)
}

func TestTagStore_SoftDelete_NotFound(t *testing.T) {
	s := datastore.NewTagStore(suite.db)

	r := &models.Tag{ID: 100}
	err := s.SoftDelete(suite.ctx, r)
	require.EqualError(t, err, "tag not found")
}

func TestTagStore_Delete(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)
	err := s.Delete(suite.ctx, 1)
	require.NoError(t, err)

	_, err = s.FindByID(suite.ctx, 1)
	require.EqualError(t, err, "tag not found")
}

func TestTagStore_Delete_NotFound(t *testing.T) {
	s := datastore.NewTagStore(suite.db)
	err := s.Delete(suite.ctx, 100)
	require.EqualError(t, err, "tag not found")
}
