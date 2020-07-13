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
	// A ManifestList has a relationship with Repository and Manifest (the insert order matters)
	testutil.ReloadFixtures(
		tb, suite.db, suite.basePath,
		testutil.RepositoriesTable, testutil.ConfigurationsTable, testutil.ManifestsTable,
		testutil.RepositoryManifestsTable, testutil.ManifestListsTable, testutil.ManifestListManifestsTable,
		testutil.RepositoryManifestListsTable,
	)
}

func unloadManifestListFixtures(tb testing.TB) {
	// A ManifestList has a relationship with Repository and Manifest (the insert order matters)
	require.NoError(tb, testutil.TruncateTables(
		suite.db,
		testutil.RepositoriesTable, testutil.ConfigurationsTable, testutil.ManifestsTable,
		testutil.RepositoryManifestsTable, testutil.ManifestListsTable, testutil.ManifestListManifestsTable,
		testutil.RepositoryManifestListsTable,
	))
}

func TestManifestListStore_ImplementsReaderAndWriter(t *testing.T) {
	require.Implements(t, (*datastore.ManifestListStore)(nil), datastore.NewManifestListStore(suite.db))
}

func TestManifestListStore_FindByID(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)

	c, err := s.FindByID(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/manifest_lists.sql
	expected := &models.ManifestList{
		ID:            1,
		SchemaVersion: 2,
		MediaType:     manifestlist.MediaTypeManifestList,
		Digest:        "sha256:dc27c897a7e24710a2821878456d56f3965df7cc27398460aa6f21f8b385d2d0",
		Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
		CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:03.470711", c.CreatedAt.Location()),
	}
	require.Equal(t, expected, c)
}

func TestManifestListStore_FindByID_NotFound(t *testing.T) {
	s := datastore.NewManifestListStore(suite.db)

	r, err := s.FindByID(suite.ctx, 0)
	require.Nil(t, r)
	require.NoError(t, err)
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
			SchemaVersion: 2,
			MediaType:     manifestlist.MediaTypeManifestList,
			Digest:        "sha256:dc27c897a7e24710a2821878456d56f3965df7cc27398460aa6f21f8b385d2d0",
			Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:03.470711", local),
		},
		{
			ID:            2,
			SchemaVersion: 2,
			MediaType:     manifestlist.MediaTypeManifestList,
			Digest:        "sha256:45e85a20d32f249c323ed4085026b6b0ee264788276aa7c06cf4b5da1669067a",
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

func TestManifestListStore_Manifests(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	mm, err := s.Manifests(suite.ctx, &models.ManifestList{ID: 1})
	require.NoError(t, err)

	// see testdata/fixtures/manifest_list_manifests.sql
	local := mm[0].CreatedAt.Location()
	expected := models.Manifests{
		{
			ID:              1,
			ConfigurationID: sql.NullInt64{Int64: 1, Valid: true},
			SchemaVersion:   2,
			MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
			Digest:          "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
			Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
			CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:              2,
			ConfigurationID: sql.NullInt64{Int64: 2, Valid: true},
			SchemaVersion:   2,
			MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
			Digest:          "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
			Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
			CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
	}
	require.Equal(t, expected, mm)
}

func TestManifestListStore_Repositories(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	rr, err := s.Repositories(suite.ctx, &models.ManifestList{ID: 1})
	require.NoError(t, err)

	// see testdata/fixtures/repository_manifest_lists.sql
	local := rr[0].CreatedAt.Location()
	expected := models.Repositories{
		{
			ID:        3,
			Name:      "backend",
			Path:      "gitlab-org/gitlab-test/backend",
			ParentID:  sql.NullInt64{Int64: 2, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:42:12.566212", local),
		},
		{
			ID:        7,
			Name:      "bar",
			Path:      "a-test-group/bar",
			ParentID:  sql.NullInt64{Int64: 5, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-06-08 16:01:39.476421", local),
		},
	}
	require.Equal(t, expected, rr)
}

func TestManifestListStore_Create(t *testing.T) {
	unloadManifestListFixtures(t)
	reloadRepositoryFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	c := &models.ManifestList{
		SchemaVersion: 2,
		MediaType:     manifestlist.MediaTypeManifestList,
		Digest:        "sha256:83501e939d64ac48ff57077efb8cc6a232aad28a0aa907a1a2b61144cb677207",
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
		SchemaVersion: 2,
		MediaType:     manifestlist.MediaTypeManifestList,
		Digest:        "sha256:83501e939d64ac48ff57077efb8cc6a232aad28a0aa907a1a2b61144cb677207",
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
		SchemaVersion: 2,
		MediaType:     manifestlist.MediaTypeManifestList,
		Digest:        "sha256:83501e939d64ac48ff57077efb8cc6a232aad28a0aa907a1a2b61144cb677207",
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

func TestManifestListStore_AssociateManifest(t *testing.T) {
	reloadManifestListFixtures(t)
	require.NoError(t, testutil.TruncateTables(suite.db, testutil.ManifestListManifestsTable))

	s := datastore.NewManifestListStore(suite.db)

	// see testdata/fixtures/manifest_list_manifests.sql
	ml := &models.ManifestList{ID: 1}
	m := &models.Manifest{ID: 3}
	err := s.AssociateManifest(suite.ctx, ml, m)
	require.NoError(t, err)

	mm, err := s.Manifests(suite.ctx, ml)
	require.NoError(t, err)

	var assocManifestIDs []int64
	for _, m := range mm {
		assocManifestIDs = append(assocManifestIDs, m.ID)
	}
	require.Contains(t, assocManifestIDs, int64(3))
}

func TestManifestListStore_AssociateManifest_AlreadyAssociatedDoesNotFail(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)

	// see testdata/fixtures/manifest_list_manifests.sql
	ml := &models.ManifestList{ID: 1}
	m := &models.Manifest{ID: 1}
	err := s.AssociateManifest(suite.ctx, ml, m)
	require.NoError(t, err)
}

func TestManifestListStore_DissociateManifest(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	ml := &models.ManifestList{ID: 1}
	m := &models.Manifest{ID: 1}

	err := s.DissociateManifest(suite.ctx, ml, m)
	require.NoError(t, err)

	mm, err := s.Manifests(suite.ctx, ml)
	require.NoError(t, err)

	// see testdata/fixtures/manifest_list_manifests.sql
	var manifestIDs []int64
	for _, m := range mm {
		manifestIDs = append(manifestIDs, m.ID)
	}
	require.NotContains(t, manifestIDs, 1)
}

func TestManifestListStore_DissociateManifest_NotAssociatedDoesNotFail(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	ml := &models.ManifestList{ID: 1}
	m := &models.Manifest{ID: 3}

	err := s.DissociateManifest(suite.ctx, ml, m)
	require.NoError(t, err)
}

func TestManifestListStore_Delete(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewManifestListStore(suite.db)
	err := s.Delete(suite.ctx, 1)
	require.NoError(t, err)

	ml, err := s.FindByID(suite.ctx, 1)
	require.Nil(t, ml)
}

func TestManifestListStore_Delete_NotFound(t *testing.T) {
	s := datastore.NewManifestListStore(suite.db)
	err := s.Delete(suite.ctx, 100)
	require.EqualError(t, err, "manifest list not found")
}
