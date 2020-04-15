// +build integration

package datastore_test

import (
	"database/sql"
	"encoding/json"
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
		testutil.RepositoriesTable, testutil.ManifestConfigurationsTable, testutil.ManifestsTable, testutil.TagsTable,
	)
}

func unloadTagFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(
		suite.db,
		// A Tag has a foreign key for a Manifest, which in turn references a Repository (insert order matters)
		testutil.RepositoriesTable, testutil.ManifestConfigurationsTable, testutil.ManifestsTable, testutil.TagsTable,
	))
}

func TestTagStore_FindByID(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	tag, err := s.FindByID(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	expected := &models.Tag{
		ID:           1,
		Name:         "1.0.0",
		RepositoryID: 3,
		ManifestID:   1,
		CreatedAt:    testutil.ParseTimestamp(t, "2020-03-02 17:57:43.283783", tag.CreatedAt.Location()),
	}
	require.Equal(t, expected, tag)
}

func TestTagStore_FindByID_NotFound(t *testing.T) {
	s := datastore.NewTagStore(suite.db)

	r, err := s.FindByID(suite.ctx, 0)
	require.Nil(t, r)
	require.EqualError(t, err, "tag not found")
}

func TestTagStore_FindByNameAndRepositoryID(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	tag, err := s.FindByNameAndRepositoryID(suite.ctx, "2.0.0", 3)
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	excepted := &models.Tag{
		ID:           2,
		Name:         "2.0.0",
		RepositoryID: 3,
		ManifestID:   2,
		CreatedAt:    testutil.ParseTimestamp(t, "2020-03-02 17:57:44.283783", tag.CreatedAt.Location()),
	}
	require.Equal(t, excepted, tag)
}

func TestTagStore_FindByNameAndRepositoryID_NotFound(t *testing.T) {
	s := datastore.NewTagStore(suite.db)

	tag, err := s.FindByNameAndRepositoryID(suite.ctx, "3.0.0", 3)
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
			ID:           1,
			Name:         "1.0.0",
			RepositoryID: 3,
			ManifestID:   1,
			CreatedAt:    testutil.ParseTimestamp(t, "2020-03-02 17:57:43.283783", local),
		},
		{
			ID:           2,
			Name:         "2.0.0",
			RepositoryID: 3,
			ManifestID:   2,
			CreatedAt:    testutil.ParseTimestamp(t, "2020-03-02 17:57:44.283783", local),
		},
		{
			ID:           3,
			Name:         "latest",
			RepositoryID: 3,
			ManifestID:   2,
			CreatedAt:    testutil.ParseTimestamp(t, "2020-03-02 17:57:45.283783", local),
			UpdatedAt: sql.NullTime{
				Time:  testutil.ParseTimestamp(t, "2020-03-02 17:57:53.029514", local),
				Valid: true,
			},
		},
		{
			ID:           4,
			Name:         "1.0.0",
			RepositoryID: 4,
			ManifestID:   3,
			CreatedAt:    testutil.ParseTimestamp(t, "2020-03-02 17:57:46.283783", local),
		},
		{
			ID:           5,
			Name:         "latest",
			RepositoryID: 4,
			ManifestID:   3,
			CreatedAt:    testutil.ParseTimestamp(t, "2020-03-02 17:57:47.283783", local),
		},
		{
			ID:           6,
			Name:         "0.0.1",
			RepositoryID: 4,
			ManifestID:   4,
			CreatedAt:    testutil.ParseTimestamp(t, "2020-04-15 09:47:26.461413", local),
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
	require.Equal(t, 6, count)
}

func TestTagStore_Repository(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	r, err := s.Repository(suite.ctx, &models.Tag{ID: 2, RepositoryID: 3})
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	excepted := &models.Repository{
		ID:        3,
		Name:      "backend",
		Path:      "gitlab-org/gitlab-test/backend",
		ParentID:  sql.NullInt64{Int64: 2, Valid: true},
		CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:42:12.566212", r.CreatedAt.Location()),
	}
	require.Equal(t, excepted, r)
}

func TestTagStore_Manifest(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)

	m, err := s.Manifest(suite.ctx, &models.Tag{ID: 2, ManifestID: 2})
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	excepted := &models.Manifest{
		ID:              2,
		SchemaVersion:   2,
		MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
		Digest:          "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
		ConfigurationID: sql.NullInt64{Int64: 2, Valid: true},
		Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
		CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", m.CreatedAt.Location()),
	}
	require.Equal(t, excepted, m)
}

func TestTagStore_Create(t *testing.T) {
	unloadTagFixtures(t)
	reloadRepositoryFixtures(t)
	reloadManifestFixtures(t)

	s := datastore.NewTagStore(suite.db)
	tag := &models.Tag{
		Name:         "3.0.0",
		RepositoryID: 3,
		ManifestID:   1,
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
		Name:         "1.0.0",
		RepositoryID: 3,
		ManifestID:   1,
	}
	err := s.Create(suite.ctx, tag)
	require.Error(t, err)
}

func TestTagStore_Update(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewTagStore(suite.db)
	update := &models.Tag{
		ID:           5,
		Name:         "2.0.0",
		RepositoryID: 4,
		ManifestID:   1,
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
		ID:           100,
		Name:         "foo",
		RepositoryID: 4,
		ManifestID:   1,
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
