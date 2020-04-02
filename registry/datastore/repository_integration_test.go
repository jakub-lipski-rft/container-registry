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
