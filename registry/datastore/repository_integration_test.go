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

func reloadRepositoryFixtures(tb testing.TB) {
	testutil.ReloadFixtures(tb, suite.db, suite.basePath, testutil.RepositoriesTable)
}

func unloadRepositoryFixtures(tb testing.TB) {
	require.NoError(tb, testutil.TruncateTables(suite.db, testutil.RepositoriesTable))
}

func TestRepositoryStore_FindByID(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	r, err := s.FindByID(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	excepted := &models.Repository{
		ID:        1,
		Name:      "gitlab-org",
		Path:      "gitlab-org",
		CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:47:39.849864", r.CreatedAt.Location()),
	}
	require.Equal(t, excepted, r)
}

func TestRepositoryStore_FindByID_NotFound(t *testing.T) {
	s := datastore.NewRepositoryStore(suite.db)
	r, err := s.FindByID(suite.ctx, 0)
	require.Nil(t, r)
	require.EqualError(t, err, "repository not found")
}

func TestRepositoryStore_FindByPath(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	r, err := s.FindByPath(suite.ctx, "gitlab-org/gitlab-test")
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	excepted := &models.Repository{
		ID:        2,
		Name:      "gitlab-test",
		Path:      "gitlab-org/gitlab-test",
		ParentID:  sql.NullInt64{Int64: 1, Valid: true},
		CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:47:40.866312", r.CreatedAt.Location()),
	}
	require.Equal(t, excepted, r)
}

func TestRepositoryStore_FindByPath_NotFound(t *testing.T) {
	s := datastore.NewRepositoryStore(suite.db)
	r, err := s.FindByPath(suite.ctx, "foo/bar")
	require.Nil(t, r)
	require.EqualError(t, err, "repository not found")
}

func TestRepositoryStore_All(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	require.Len(t, rr, 4)
	local := rr[0].CreatedAt.Location()
	expected := models.Repositories{
		{
			ID:        1,
			Name:      "gitlab-org",
			Path:      "gitlab-org",
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:47:39.849864", local),
		},
		{
			ID:        2,
			Name:      "gitlab-test",
			Path:      "gitlab-org/gitlab-test",
			ParentID:  sql.NullInt64{Int64: 1, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:47:40.866312", local),
		},
		{
			ID:        3,
			Name:      "backend",
			Path:      "gitlab-org/gitlab-test/backend",
			ParentID:  sql.NullInt64{Int64: 2, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:42:12.566212", local),
		},
		{
			ID:        4,
			Name:      "frontend",
			Path:      "gitlab-org/gitlab-test/frontend",
			ParentID:  sql.NullInt64{Int64: 2, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:43:39.476421", local),
		},
	}

	require.Equal(t, expected, rr)
}

func TestRepositoryStore_All_NotFound(t *testing.T) {
	unloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindAll(suite.ctx)
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestRepositoryStore_DescendantsOf(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindDescendantsOf(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	local := rr[0].CreatedAt.Location()
	expected := models.Repositories{
		{
			ID:        2,
			Name:      "gitlab-test",
			Path:      "gitlab-org/gitlab-test",
			ParentID:  sql.NullInt64{Int64: 1, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:47:40.866312", local),
		},
		{
			ID:        3,
			Name:      "backend",
			Path:      "gitlab-org/gitlab-test/backend",
			ParentID:  sql.NullInt64{Int64: 2, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:42:12.566212", local),
		},
		{
			ID:        4,
			Name:      "frontend",
			Path:      "gitlab-org/gitlab-test/frontend",
			ParentID:  sql.NullInt64{Int64: 2, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:43:39.476421", local),
		},
	}

	require.Equal(t, expected, rr)
}

func TestRepositoryStore_DescendantsOf_Leaf(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindDescendantsOf(suite.ctx, 3)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestRepositoryStore_DescendantsOf_NotFound(t *testing.T) {
	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindDescendantsOf(suite.ctx, 0)
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestRepositoryStore_AncestorsOf(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindAncestorsOf(suite.ctx, 3)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	local := rr[0].CreatedAt.Location()
	expected := models.Repositories{
		{
			ID:        2,
			Name:      "gitlab-test",
			Path:      "gitlab-org/gitlab-test",
			ParentID:  sql.NullInt64{Int64: 1, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:47:40.866312", local),
		},
		{
			ID:        1,
			Name:      "gitlab-org",
			Path:      "gitlab-org",
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:47:39.849864", local),
		},
	}

	require.Equal(t, expected, rr)
}

func TestRepositoryStore_AncestorsOf_Root(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindAncestorsOf(suite.ctx, 1)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestRepositoryStore_AncestorsOf_NotFound(t *testing.T) {
	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindAncestorsOf(suite.ctx, 0)
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestRepositoryStore_SiblingsOf(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindSiblingsOf(suite.ctx, 3)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	local := rr[0].CreatedAt.Location()
	expected := models.Repositories{
		{
			ID:        4,
			Name:      "frontend",
			Path:      "gitlab-org/gitlab-test/frontend",
			ParentID:  sql.NullInt64{Int64: 2, Valid: true},
			CreatedAt: testutil.ParseTimestamp(t, "2020-03-02 17:43:39.476421", local),
		},
	}

	require.Equal(t, expected, rr)
}

func TestRepositoryStore_SiblingsOf_OnlyChild(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindSiblingsOf(suite.ctx, 2)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	require.Len(t, rr, 0)
}

func TestRepositoryStore_SiblingsOf_NotFound(t *testing.T) {
	s := datastore.NewRepositoryStore(suite.db)
	rr, err := s.FindSiblingsOf(suite.ctx, 0)
	require.Empty(t, rr)
	require.NoError(t, err)
}

func TestRepositoryStore_Manifests(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	mm, err := s.Manifests(suite.ctx, &models.Repository{ID: 3})
	require.NoError(t, err)

	// see testdata/fixtures/repository_manifests.sql
	local := mm[0].CreatedAt.Location()
	expected := models.Manifests{
		{
			ID:              1,
			SchemaVersion:   2,
			MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
			Digest:          "sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155",
			ConfigurationID: 1,
			Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1640,"digest":"sha256:ea8a54fd13889d3649d0a4e45735116474b8a650815a2cda4940f652158579b9"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"}]}`),
			CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
		{
			ID:              2,
			SchemaVersion:   2,
			MediaType:       "application/vnd.docker.distribution.manifest.v2+json",
			Digest:          "sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f",
			ConfigurationID: 2,
			Payload:         json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":1819,"digest":"sha256:9ead3a93fc9c9dd8f35221b1f22b155a513815b7b00425d6645b34d98e83b073"},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":2802957,"digest":"sha256:c9b1b535fdd91a9855fb7f82348177e5f019329a58c53c47272962dd60f71fc9"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":108,"digest":"sha256:6b0937e234ce911b75630b744fb12836fe01bda5f7db203927edbb1390bc7e21"},{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":109,"digest":"sha256:f01256086224ded321e042e74135d72d5f108089a1cda03ab4820dfc442807c1"}]}`),
			CreatedAt:       testutil.ParseTimestamp(t, "2020-03-02 17:50:26.461745", local),
		},
	}
	require.Equal(t, expected, mm)
}

func TestRepositoryStore_ManifestLists(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	mm, err := s.ManifestLists(suite.ctx, &models.Repository{ID: 3})
	require.NoError(t, err)

	// see testdata/fixtures/repository_manifest_lists.sql
	local := mm[0].CreatedAt.Location()
	expected := models.ManifestLists{
		{
			ID:            1,
			SchemaVersion: 2,
			MediaType:     sql.NullString{String: manifestlist.MediaTypeManifestList, Valid: true},
			Payload:       json.RawMessage(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json","manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":23321,"digest":"sha256:bd165db4bd480656a539e8e00db265377d162d6b98eebbfe5805d0fbd5144155","platform":{"architecture":"amd64","os":"linux"}},{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":24123,"digest":"sha256:56b4b2228127fd594c5ab2925409713bd015ae9aa27eef2e0ddd90bcb2b1533f","platform":{"architecture":"amd64","os":"windows","os.version":"10.0.14393.2189"}}]}`),
			CreatedAt:     testutil.ParseTimestamp(t, "2020-04-02 18:45:03.470711", local),
		},
	}
	require.Equal(t, expected, mm)
}

func TestRepositoryStore_Tags(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	tt, err := s.Tags(suite.ctx, &models.Repository{ID: 4})
	require.NoError(t, err)

	// see testdata/fixtures/tags.sql
	local := tt[0].CreatedAt.Location()
	expected := models.Tags{
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
	}
	require.Equal(t, expected, tt)
}

func TestRepositoryStore_ManifestTags(t *testing.T) {
	reloadTagFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	tt, err := s.ManifestTags(suite.ctx, &models.Repository{ID: 3}, &models.Manifest{ID: 1})
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
	}
	require.Equal(t, expected, tt)
}

func TestRepositoryStore_Count(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	count, err := s.Count(suite.ctx)
	require.NoError(t, err)

	// see testdata/fixtures/repositories.sql
	require.Equal(t, 4, count)
}

func TestRepositoryStore_Create(t *testing.T) {
	unloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	r := &models.Repository{
		Name: "bar",
		Path: "foo/bar",
	}
	err := s.Create(suite.ctx, r)

	require.NoError(t, err)
	require.NotEmpty(t, r.ID)
	require.NotEmpty(t, r.CreatedAt)
}

func TestRepositoryStore_Create_NonUniquePathFails(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	r := &models.Repository{
		Name:     "gitlab-test",
		Path:     "gitlab-org/gitlab-test",
		ParentID: sql.NullInt64{Int64: 1, Valid: true},
	}
	err := s.Create(suite.ctx, r)
	require.Error(t, err)
}

func TestRepositoryStore_Update(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	update := &models.Repository{
		ID:       4,
		Name:     "bar",
		Path:     "bar",
		ParentID: sql.NullInt64{Int64: 0, Valid: false},
	}
	err := s.Update(suite.ctx, update)
	require.NoError(t, err)

	r, err := s.FindByID(suite.ctx, update.ID)
	require.NoError(t, err)

	update.CreatedAt = r.CreatedAt
	require.Equal(t, update, r)
}

func TestRepositoryStore_Update_NotFound(t *testing.T) {
	s := datastore.NewRepositoryStore(suite.db)

	update := &models.Repository{
		ID:   5,
		Name: "bar",
	}
	err := s.Update(suite.ctx, update)
	require.EqualError(t, err, "repository not found")
}

func TestRepositoryStore_AssociateManifest(t *testing.T) {
	reloadManifestFixtures(t)
	require.NoError(t, testutil.TruncateTables(suite.db, testutil.RepositoryManifestsTable))

	s := datastore.NewRepositoryStore(suite.db)
	// see testdata/fixtures/repository_manifests.sql
	r := &models.Repository{ID: 4}
	m := &models.Manifest{ID: 2}

	err := s.AssociateManifest(suite.ctx, r, m)
	require.NoError(t, err)

	mm, err := s.Manifests(suite.ctx, r)
	require.NoError(t, err)

	var assocManifestIDs []int
	for _, m := range mm {
		assocManifestIDs = append(assocManifestIDs, m.ID)
	}
	require.Contains(t, assocManifestIDs, 2)
}

func TestRepositoryStore_AssociateManifest_AlreadyAssociatedDoesNotFail(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	// see testdata/fixtures/repository_manifests.sql
	r := &models.Repository{ID: 3}
	m := &models.Manifest{ID: 1}
	err := s.AssociateManifest(suite.ctx, r, m)
	require.NoError(t, err)
}

func TestRepositoryStore_DissociateManifest(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	// see testdata/fixtures/repository_manifests.sql
	r := &models.Repository{ID: 3}
	m := &models.Manifest{ID: 1}

	err := s.DissociateManifest(suite.ctx, r, m)
	require.NoError(t, err)

	mm, err := s.Manifests(suite.ctx, r)
	require.NoError(t, err)

	for _, m := range mm {
		require.NotEqual(t, 1, m.ID)
	}
}

func TestRepositoryStore_DissociateManifest_NotAssociatedDoesNotFail(t *testing.T) {
	reloadManifestFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	// see testdata/fixtures/repository_manifests.sql
	r := &models.Repository{ID: 4}
	m := &models.Manifest{ID: 1}
	err := s.DissociateManifest(suite.ctx, r, m)
	require.NoError(t, err)
}

func TestRepositoryStore_AssociateManifestList(t *testing.T) {
	reloadManifestListFixtures(t)
	require.NoError(t, testutil.TruncateTables(suite.db, testutil.RepositoryManifestListsTable))

	s := datastore.NewRepositoryStore(suite.db)
	// see testdata/fixtures/repository_manifest_lists.sql
	r := &models.Repository{ID: 4}
	ml := &models.ManifestList{ID: 2}

	err := s.AssociateManifestList(suite.ctx, r, ml)
	require.NoError(t, err)

	ll, err := s.ManifestLists(suite.ctx, r)
	require.NoError(t, err)

	var assocManifestListIDs []int
	for _, ml := range ll {
		assocManifestListIDs = append(assocManifestListIDs, ml.ID)
	}
	require.Contains(t, assocManifestListIDs, 2)
}

func TestRepositoryStore_AssociateManifestList_AlreadyAssociatedDoesNotFail(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	// see testdata/fixtures/repository_manifest_lists.sql
	r := &models.Repository{ID: 3}
	ml := &models.ManifestList{ID: 1}
	err := s.AssociateManifestList(suite.ctx, r, ml)
	require.NoError(t, err)
}

func TestRepositoryStore_DissociateManifestList(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	// see testdata/fixtures/repository_manifest_lists.sql
	r := &models.Repository{ID: 3}
	ml := &models.ManifestList{ID: 1}

	err := s.DissociateManifestList(suite.ctx, r, ml)
	require.NoError(t, err)

	ll, err := s.ManifestLists(suite.ctx, r)
	require.NoError(t, err)

	for _, ml := range ll {
		require.NotEqual(t, 1, ml.ID)
	}
}

func TestRepositoryStore_DissociateManifestList_NotAssociatedDoesNotFail(t *testing.T) {
	reloadManifestListFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	// see testdata/fixtures/repository_manifest_lists.sql
	r := &models.Repository{ID: 4}
	ml := &models.ManifestList{ID: 1}
	err := s.DissociateManifestList(suite.ctx, r, ml)
	require.NoError(t, err)
}

func TestRepositoryStore_SoftDelete(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)

	r := &models.Repository{ID: 4}
	err := s.SoftDelete(suite.ctx, r)
	require.NoError(t, err)

	r, err = s.FindByID(suite.ctx, r.ID)
	require.NoError(t, err)

	require.True(t, r.DeletedAt.Valid)
	require.NotEmpty(t, r.DeletedAt.Time)
}

func TestRepositoryStore_SoftDelete_NotFound(t *testing.T) {
	s := datastore.NewRepositoryStore(suite.db)

	r := &models.Repository{ID: 5}
	err := s.SoftDelete(suite.ctx, r)
	require.EqualError(t, err, "repository not found")
}

func TestRepositoryStore_Delete(t *testing.T) {
	reloadRepositoryFixtures(t)

	s := datastore.NewRepositoryStore(suite.db)
	err := s.Delete(suite.ctx, 4)
	require.NoError(t, err)

	_, err = s.FindByID(suite.ctx, 4)
	require.EqualError(t, err, "repository not found")
}

func TestRepositoryStore_Delete_NotFound(t *testing.T) {
	s := datastore.NewRepositoryStore(suite.db)
	err := s.Delete(suite.ctx, 5)
	require.EqualError(t, err, "repository not found")
}
