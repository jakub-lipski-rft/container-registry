// +build integration

package datastore_test

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/docker/distribution/manifest/manifestlist"

	"github.com/docker/distribution/registry/datastore"

	"github.com/docker/distribution/registry/datastore/models"
	"github.com/docker/distribution/registry/datastore/testutil"
	"github.com/stretchr/testify/require"
)

func reloadManifestListFixtures(tb testing.TB) {
	// A ManifestList has a foreign key for a Repository (the insert order matters)
	testutil.ReloadFixtures(tb, suite.db, suite.basePath, testutil.RepositoriesTable, testutil.ManifestListsTable)
}

func unloadManifestListFixtures(tb testing.TB) {
	// A ManifestList has a foreign key for a Repository (the insert order matters)
	require.NoError(tb, testutil.TruncateTables(suite.db, testutil.RepositoriesTable, testutil.ManifestListsTable))
}

func TestManifestListStore_FindByID(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)

	c, err := s.FindByID(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/manifest_lists.sql
	expected := &models.ManifestList{
		ID:            1,
		RepositoryID:  3,
		SchemaVersion: 2,
		MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
		Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
		CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:03.470711", c.CreatedAt.Location()),
	}
	require.Equal(t, expected, c)
}

func TestManifestListStore_FindByID_NotFound(t *testing.T) {
	s := datastore.NewManifestListStore(suite.db)

	r, err := s.FindByID(suite.ctx, 0)
	require.Nil(t, r)
	require.EqualError(t, err, "manifest list not found")
}

func TestManifestListStore_FindAll(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)

	cc, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/manifest_lists.sql
	local := cc[0].CreatedAt.Location()
	expected := models.ManifestLists{
		{
			ID:            1,
			RepositoryID:  3,
			SchemaVersion: 2,
			MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
			Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:03.470711", local),
		},
		{
			ID:            2,
			RepositoryID:  4,
			SchemaVersion: 2,
			MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
			Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":42212,"digest":"sha256:bca3c0bf2ca0cde987ad9cab2dac986047a0ccff282f1b23df282ef05e3a10a6","platform":{"architecture":"amd64","os":"linux"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:04.470711", local),
		},
	}
	require.Equal(t, expected, cc)
}

func TestManifestListStore_FindAll_NotFound(t *testing.T) {
	unloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)

	cc, err := s.FindAll(suite.ctx)
	require.Empty(t, cc)
	require.NoError(t, err)
}

func TestManifestListStore_Count(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/manifest_lists.sql
	require.Equal(t, 2, count)
}

func TestManifestListStore_Create(t *testing.T) {
	unloadManifestListFixtures(t)
	reloadRepositoryFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	c := &models.ManifestList{
		RepositoryID:  3,
		SchemaVersion: 2,
		MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
		Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
	}
	err := s.Create(suite.ctx, c)

	require.NoError(t, err)
	require.NotEmpty(t, c.ID)
	require.NotEmpty(t, c.CreatedAt)
}

func TestManifestListStore_Update(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	update := &models.ManifestList{
		ID:            1,
		RepositoryID:  4,
		SchemaVersion: 2,
		MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
		Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":42212,"digest":"sha256:bca3c0bf2ca0cde987ad9cab2dac986047a0ccff282f1b23df282ef05e3a10a6","platform":{"architecture":"amd64","os":"linux"}}]}`),
	}
	err := s.Update(suite.ctx, update)
	require.NoError(t, err)

	r, err := s.FindByID(suite.ctx, update.ID)
	require.NoError(t, err)

	update.CreatedAt = r.CreatedAt
	require.Equal(t, update, r)
}

func TestManifestListStore_Update_NotFound(t *testing.T) {
	s := datastore.NewManifestListStore(suite.db)

	update := &models.ManifestList{
		ID:            100,
		RepositoryID:  4,
		SchemaVersion: 2,
		MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
		Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"...","config":{}}`),
	}

	err := s.Update(suite.ctx, update)
	require.EqualError(t, err, "manifest list not found")
}

func TestManifestListStore_Mark(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)

	r := &models.ManifestList{ID: 1}
	err := s.Mark(suite.ctx, r)
	require.NoError(t, err)

	require.True(t, r.MarkedAt.Valid)
	require.NotEmpty(t, r.MarkedAt.Time)
}

func TestManifestListStore_Mark_NotFound(t *testing.T) {
	s := datastore.NewManifestListStore(suite.db)

	r := &models.ManifestList{ID: 100}
	err := s.Mark(suite.ctx, r)
	require.EqualError(t, err, "manifest list not found")
}

func TestManifestListStore_SoftDelete(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)

	r := &models.ManifestList{ID: 1}
	err := s.SoftDelete(suite.ctx, r)
	require.NoError(t, err)

	r, err = s.FindByID(suite.ctx, r.ID)
	require.NoError(t, err)

	require.True(t, r.DeletedAt.Valid)
	require.NotEmpty(t, r.DeletedAt.Time)
}

func TestManifestListStore_SoftDelete_NotFound(t *testing.T) {
	s := datastore.NewManifestListStore(suite.db)

	r := &models.ManifestList{ID: 100}
	err := s.SoftDelete(suite.ctx, r)
	require.EqualError(t, err, "manifest list not found")
}

func TestManifestListStore_Delete(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	err := s.Delete(suite.ctx, 1)
	require.NoError(t, err)

	_, err = s.FindByID(suite.ctx, 1)
	require.EqualError(t, err, "manifest list not found")
}

func TestManifestListStore_Delete_NotFound(t *testing.T) {
	s := datastore.NewManifestListStore(suite.db)
	err := s.Delete(suite.ctx, 100)
	require.EqualError(t, err, "manifest list not found")
}
